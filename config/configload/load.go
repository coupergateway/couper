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
	couperConfig := &config.Couper{
		Bytes:       src,
		Context:     eval.NewENVContext(src),
		Definitions: &config.Definitions{},
		Settings:    &defaults,
	}

	schema, _ := gohcl.ImpliedBodySchema(couperConfig)
	content, diags := body.Content(schema)
	if content == nil {
		return nil, fmt.Errorf("invalid configuration: %w", diags)
	}

	// Read possible reference definitions first. Those are the
	// base for refinement merges during server block read out.
	backends, err := loadDefinitions(couperConfig, content.Blocks)
	if err != nil {
		return nil, err
	}

	err = loadSettings(couperConfig, content.Blocks)
	if err != nil {
		return nil, err
	}

	for _, serverBlock := range content.Blocks.OfType(server) {
		serverConfig := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, couperConfig.Context, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		serverBody, err := mergeBodys(backends, serverConfig)
		if err != nil {
			return nil, err
		}

		serverConfig.Remain = serverBody

		for i, apiConfig := range serverConfig.APIs {
			apiBody, err := mergeBodys(backends, apiConfig)
			if err != nil {
				return nil, err
			}

			apiBody = MergeBodies([]hcl.Body{serverBody, apiBody})

			serverConfig.APIs[i].Remain = apiBody

			err = refineEndpoints(backends, apiBody, apiConfig.Endpoints)
			if err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		err = refineEndpoints(backends, serverBody, serverConfig.Endpoints)
		if err != nil {
			return nil, err
		}

		couperConfig.Servers = append(couperConfig.Servers, serverConfig)
	}

	if len(couperConfig.Servers) == 0 {
		return nil, fmt.Errorf("configuration error: missing server definition")
	}

	return couperConfig, nil
}

// getReference tries to fetch a backend from `definitions`
// block by a reference name, e.g. `backend = "name"`.
func getReference(
	definitionsBackends Backends, inline config.Inline,
) (hcl.Body, error) {
	reference, err := definitionsBackends.WithName(inline.Reference())
	if err != nil {
		// Backend reference is given, but not defined in definitions.
		r := inline.HCLBody().MissingItemRange()
		err.(hcl.Diagnostics)[0].Subject = &r
	}

	if reference == nil {
		reference = EmptyBody()
	}

	return reference, err
}

// mergeBodys guarantees to return a non-nil <hcl.Body> in non-error cases.
func mergeBodys(
	definitionsBackends Backends, inline config.Inline,
) (hcl.Body, error) {
	reference, err := getReference(definitionsBackends, inline)
	if err != nil {
		return nil, err
	}

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}
	if content == nil {
		return reference, nil
	}

	// Apply current attributes to the referenced body.
	if len(content.Attributes) > 0 {
		reference = MergeBodies([]hcl.Body{reference, body.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		})})
	}

	var backendBlock *hcl.Block
	if backends := content.Blocks.OfType(backend); len(backends) > 0 {
		backendBlock = backends[0]
	} else {
		fmt.Printf("mergeBodys no backends[0] %#v\n", reference)
		return reference, nil
	}

	// Case: `backend {}`, anonymous backend.
	if len(backendBlock.Labels) == 0 {
		return MergeBodies([]hcl.Body{reference, backendBlock.Body}), nil
	}

	// Case: `backend "reference" {}`, referenced backend.
	ref, err := definitionsBackends.WithName(backendBlock.Labels[0])
	if err != nil {
		err.(hcl.Diagnostics)[0].Subject = &backendBlock.DefRange

		// Case: referenced backend is not defined in definitions.
		return nil, err
	}

	if syntaxBody, ok := backendBlock.Body.(*hclsyntax.Body); ok {
		if refBody, ok := ref.(*hclsyntax.Body); ok {
			syntaxBody.Attributes[backendLabel] = refBody.Attributes[backendLabel]
		}
	}

	return MergeBodies([]hcl.Body{reference, ref}), nil
}

func refineEndpoints(
	definitionsBackends Backends, parentBody hcl.Body, endpoints config.Endpoints,
) error {
	for _, endpointConfig := range endpoints {
		content, _, diags := endpointConfig.HCLBody().PartialContent(endpointConfig.Schema(true))
		if diags.HasErrors() {
			return diags
		}

		endpointBody := MergeBodies([]hcl.Body{parentBody, body.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		})})

		for j, proxyConfig := range endpointConfig.Proxies {
			proxyBody, err := mergeBodys(definitionsBackends, proxyConfig)
			if err != nil {
				return err
			}

			proxyBody = MergeBodies([]hcl.Body{endpointBody, proxyBody})

			// block, _ := getBackendBlock(proxyConfig.HCLBody())
			// if block != nil {
			// 	proxyBody = MergeBodies([]hcl.Body{proxyBody, block.Body})
			// }

			endpointConfig.Proxies[j].Remain = proxyBody
		}

		for j, reqConfig := range endpointConfig.Requests {
			reqBody, err := mergeBodys(definitionsBackends, reqConfig)
			if err != nil {
				return err
			}

			reqBody = MergeBodies([]hcl.Body{endpointBody, reqBody})
			// a, b := reqBody.JustAttributes()
			// fmt.Printf("reqBody %#v %#v\n", a, b)
			// x, y, z := reqBody.PartialContent(reqConfig.Schema(true))
			// fmt.Printf(">>>>> %#v %#v %#v\n", x, y, z)

			endpointConfig.Requests[j].Remain = reqBody
		}
	}

	return nil
}

// func refineEndpoints(backendList Backends, parents []hcl.Body, endpoints config.Endpoints) error {
// 	for e, endpoint := range endpoints {
// 		merged, err := mergeBackendBodies(backendList, endpoint)
// 		if err != nil {
// 			return err
// 		}

// 		p := parents

// 		block, label := getBackendBlock(endpoint.HCLBody())
// 		if block != nil {
// 			p = nil
// 			for _, b := range parents {
// 				attrs, _ := b.JustAttributes()
// 				if len(attrs) == 0 {
// 					continue
// 				}

// 				if attr, ok := attrs[backendLabel]; ok {
// 					val, _ := attr.Expr.Value(nil)
// 					if label != "" && seetie.ValueToString(val) == label {
// 						p = append(p, b)
// 					}

// 					continue // skip backends with other names or block is an inline one
// 				}

// 				p = append(p, b)
// 			}
// 		}

// 		merged, err = appendPathAttribute(appendUniqueBodies(p, merged...), endpoint)
// 		if err != nil {
// 			return err
// 		}

// 		endpoints[e].Remain = MergeBodies(merged)
// 		if err = validateOrigin(endpoints[e].Remain); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

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
