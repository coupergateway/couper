package configload

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/parser"
	"github.com/avenga/couper/eval"
)

const (
	backend     = "backend"
	definitions = "definitions"
	pathAttr    = "path"
	server      = "server"
	settings    = "settings"
)

func LoadFile(filePath string) (*config.CouperFile, error) {
	_, err := config.SetWorkingDirectory(filePath)
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

func LoadBytes(src []byte, filename string) (*config.CouperFile, error) {
	hclBody, diags := parser.Load(src, filename)
	if diags.HasErrors() {
		return nil, diags
	}

	return LoadConfig(hclBody, src)
}

func LoadConfig(body hcl.Body, src []byte) (*config.CouperFile, error) {
	defaults := config.DefaultSettings
	file := &config.CouperFile{
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
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{
				{Type: backend, LabelNames: []string{"name"}},
			}})

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
					if backends.WithName(name) != nil {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("Duplicate backend name: %q", name),
							Subject:  &be.LabelRanges[0],
						}}
					}
					backends = append(backends, &Backend{
						Name:   name,
						Config: be.Body,
					})
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

		file.Server = append(file.Server, srv)

		// api block(s)
		for _, apiBlock := range []*config.Api{srv.API} {
			if apiBlock == nil {
				continue
			}

			bodies, err := mergeBackendBodies(backends, apiBlock)
			if err != nil {
				return nil, err
			}

			// empty bodies would be removed with a hcl.Merge.. later on.
			if err = refineEndpoints(backends, bodies, apiBlock.Endpoints); err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		// TODO: free endpoints
		//if err := refineEndpoints(file.Definitions, nil, srv.Endpoints); err != nil {
		//	return nil, err
		//}
	}

	if len(file.Server) == 0 {
		return nil, fmt.Errorf("configuration error: missing server definition")
	}
	return file, nil
}

func mergeBackendBodies(backendList Backends, inlineBackend config.Inline) ([]hcl.Body, error) {
	reference := backendList.WithName(inlineBackend.Reference())

	content, _, diags := inlineBackend.Body().PartialContent(inlineBackend.Schema(true))
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
		bodies = append(bodies, reference)
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
			if ref := backendList.WithName(backends[0].Labels[0]); ref != nil {
				bodies = append(bodies, ref)
			}
		}

		bodies = append(bodies, backends[0].Body)
	}

	err := validateOrigin(bodies)
	return bodies, err
}

func refineEndpoints(backendList Backends, parents []hcl.Body, endpoints config.Endpoints) error {
	for e, endpoint := range endpoints {
		merged, err := mergeBackendBodies(backendList, endpoint)
		if err != nil {
			return err
		}

		merged, err = appendPathAttribute(append(parents, merged...), endpoint)
		if err != nil {
			return err
		}

		endpoints[e].Remain = MergeBodies(merged)
	}
	return nil
}

// appendPathAttribute determines if the given endpoint has child definitions which relies on references
// which 'path' attribute should be refined with the endpoints inline value.
func appendPathAttribute(bodies []hcl.Body, endpoint *config.Endpoint) ([]hcl.Body, error) {
	if len(bodies) == 0 || endpoint == nil {
		return bodies, nil
	}

	ctnt, _, diags := endpoint.Body().PartialContent(endpoint.Schema(true))
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
func validateOrigin(bodies []hcl.Body) error {
	merged := MergeBodies(bodies)
	attrs, diags := merged.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	_, ok := attrs["origin"]
	if !ok {
		bodyRange := merged.MissingItemRange()
		return hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &bodyRange,
			Summary: "missing backend.origin attribute",
		}}
	}
	return nil
}
