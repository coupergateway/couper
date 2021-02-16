package configload

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/config/startup"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

const (
	backend      = "backend"
	backendLabel = "name"
	definitions  = "definitions"
	pathAttr     = "path"
	server       = "server"
	settings     = "settings"
)

func LoadFile(filePath string) (*config.Couper, error) {
	_, err := startup.SetWorkingDirectory(filePath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return LoadBytes(src, filename)
}

func LoadBytes(src []byte, filename string) (*config.Couper, error) {
	hclBody, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(hclBody, src)
}

func LoadConfig(body hcl.Body, src []byte) (*config.Couper, error) {
	defaults := config.DefaultSettings
	file := &config.Couper{
		Bytes:       src,
		Context:     eval.NewENVContext(src),
		Definitions: &config.Definitions{},
		Settings:    &defaults,
	}

	fileSchema, _ := gohcl.ImpliedBodySchema(file)
	content, diags := body.Content(fileSchema)
	if content == nil {
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	// reading possible reference definitions first. Those are the base for refinement merges during server block read out.
	var backends Backends
	for _, outerBlock := range content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)

			if diags.HasErrors() {
				return nil, diags
			}

			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					if len(be.Labels) == 0 {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Missing backend name",
							Subject:  &be.DefRange,
						}}
					}
					name := be.Labels[0]
					ref, _ := backends.WithName(name)
					if ref != nil {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("duplicate backend name: %q", name),
							Subject:  &be.LabelRanges[0],
						}}
					}
					backends = append(backends, NewBackend(name, be.Body))
				}
			}

			if diags = gohcl.DecodeBody(leftOver, file.Context, file.Definitions); diags.HasErrors() {
				return nil, diags
			}
		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, file.Context, file.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	// reading per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range content.Blocks.OfType(server) {
		srv := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, file.Context, srv); diags.HasErrors() {
			return nil, diags
		}

		if len(serverBlock.Labels) > 0 {
			srv.Name = serverBlock.Labels[0]
		}

		file.Servers = append(file.Servers, srv)

		serverBodies, err := mergeBackendBodies(backends, srv)
		if err != nil {
			return nil, err
		}
		srv.Remain = MergeBodies(serverBodies)

		// api block(s)
		for _, apiBlock := range srv.APIs {
			if apiBlock == nil {
				continue
			}

			bodies, err := mergeBackendBodies(backends, apiBlock)
			if err != nil {
				return nil, err
			}

			bodies = appendUniqueBodies(serverBodies, bodies...)

			// empty bodies would be removed with a hcl.Merge.. later on.
			if err = refineEndpoints(backends, bodies, apiBlock.Endpoints); err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		if err := refineEndpoints(backends, serverBodies, srv.Endpoints); err != nil {
			return nil, err
		}
	}

	if len(file.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing server definition")
	}
	return file, nil
}

func mergeBackendBodies(backendList Backends, inlineBackend config.Inline) ([]hcl.Body, error) {
	reference, err := backendList.WithName(inlineBackend.Reference())
	if err != nil {
		r := inlineBackend.HCLBody().MissingItemRange()
		err.(hcl.Diagnostics)[0].Subject = &r
		return nil, err
	}

	content, _, diags := inlineBackend.HCLBody().PartialContent(inlineBackend.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	var bodies []hcl.Body
	var backends hcl.Blocks
	if content != nil {
		backends = content.Blocks.OfType(backend)
	}

	if reference != nil {
		if content != nil && len(backends) > 0 {
			return nil, fmt.Errorf("configuration error: inlineBackend reference and inline definition")
		}
		// we have a reference, append to list and...
		bodies = appendUniqueBodies(bodies, reference)
	}
	// ...additionally add the inline overrides.
	if content != nil && len(content.Attributes) > 0 {
		bodies = append(bodies, body.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		}))
	}

	if len(backends) > 0 {
		if len(backends[0].Labels) > 0 {
			ref, err := backendList.WithName(backends[0].Labels[0])
			if err != nil {
				err.(hcl.Diagnostics)[0].Subject = &backends[0].DefRange
				return nil, err
			}
			// link name attribute
			if syntaxBody, ok := backends[0].Body.(*hclsyntax.Body); ok {
				if refBody, ok := ref.(*hclsyntax.Body); ok {
					syntaxBody.Attributes[backendLabel] = refBody.Attributes[backendLabel]
				}
			}
			bodies = append([]hcl.Body{ref}, bodies...)
		}

		bodies = appendUniqueBodies(bodies, backends[0].Body)
	}

	return bodies, nil
}

func refineEndpoints(backendList Backends, parents []hcl.Body, endpoints config.Endpoints) error {
	for e, endpoint := range endpoints {
		merged, err := mergeBackendBodies(backendList, endpoint)
		if err != nil {
			return err
		}

		p := parents
		block, label := getBackendBlock(endpoint.HCLBody())
		if block != nil {
			p = nil
			for _, b := range parents {
				attrs, _ := b.JustAttributes()
				if len(attrs) == 0 {
					continue
				}
				if attr, ok := attrs[backendLabel]; ok {
					val, _ := attr.Expr.Value(nil)
					if label != "" && seetie.ValueToString(val) == label {
						p = append(p, b)
					}
					continue // skip backends with other names or block is an inline one
				}
				p = append(p, b)
			}
		}

		merged, err = appendPathAttribute(appendUniqueBodies(p, merged...), endpoint)
		if err != nil {
			return err
		}

		endpoints[e].Remain = MergeBodies(merged)
		if err = validateOrigin(endpoints[e].Remain); err != nil {
			return err
		}
	}
	return nil
}

// appendPathAttribute determines if the given endpoint has child definitions which relies on references
// which 'path' attribute should be refined with the endpoints inline value.
func appendPathAttribute(bodies []hcl.Body, endpoint *config.Endpoint) ([]hcl.Body, error) {
	if len(bodies) == 0 || endpoint == nil {
		return bodies, nil
	}

	ctnt, _, diags := endpoint.HCLBody().PartialContent(endpoint.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	} else if ctnt == nil {
		return bodies, nil
	}

	// do not override path with an endpoint one if a reference override or
	// an inline definition contains an explicit path attribute value.
	blocks := ctnt.Blocks.OfType(backend)
	if len(blocks) > 0 {
		bodyAttrs, _ := blocks[0].Body.JustAttributes()
		for name := range bodyAttrs {
			if name == pathAttr {
				return bodies, nil
			}
		}
	}

	for name, attr := range ctnt.Attributes {
		if name == pathAttr {
			return append(bodies, body.New(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					pathAttr: attr,
				},
			})), nil
		}
	}

	return bodies, nil
}

// validateOrigin checks at least for an origin attribute definition.
func validateOrigin(merged hcl.Body) error {
	content, _, diags := merged.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "origin"}}})
	if diags.HasErrors() {
		return diags
	}

	_, ok := content.Attributes["origin"]
	if !ok {
		err := errors.New("missing backend.origin attribute")
		bodyRange := merged.MissingItemRange()
		if bodyRange.Filename == "<empty>" {
			return err
		}
		return hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &bodyRange,
			Summary: err.Error(),
		}}
	}
	return nil
}

func appendUniqueBodies(parents []hcl.Body, bodies ...hcl.Body) []hcl.Body {
	merged := parents[:]
	for _, b := range bodies {
		unique := true
		for _, m := range merged {
			if m == b {
				unique = false
				break
			}
		}
		if unique {
			merged = append(merged, b)
		}
	}
	return merged
}
