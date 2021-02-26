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
	var definedBackends Backends

	for _, outerBlock := range content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return nil, diags
			}

			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					name := be.Labels[0]
					ref, _ := definedBackends.WithName(name)
					if ref != nil {
						return nil, hcl.Diagnostics{&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("duplicate backend name: %q", name),
							Subject:  &be.LabelRanges[0],
						}}
					}

					definedBackends = append(definedBackends, NewBackend(name, be.Body))
				}
			}

			if diags = gohcl.DecodeBody(leftOver, couperConfig.Context, couperConfig.Definitions); diags.HasErrors() {
				return nil, diags
			}
		case settings:
			if diags = gohcl.DecodeBody(outerBlock.Body, couperConfig.Context, couperConfig.Settings); diags.HasErrors() {
				return nil, diags
			}
		}
	}

	// Read per server block and merge backend settings which results in a final server configuration.
	for _, serverBlock := range content.Blocks.OfType(server) {
		serverConfig := &config.Server{}
		if diags = gohcl.DecodeBody(serverBlock.Body, couperConfig.Context, serverConfig); diags.HasErrors() {
			return nil, diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		// Read server inline, reference overrides or referenced backends
		serverBackend, mergeErr := mergeBackendBodies(definedBackends, serverConfig)
		if mergeErr != nil {
			return nil, mergeErr
		}

		// Read api blocks and merge backends with server and definitions backends.
		for _, apiBlock := range serverConfig.APIs {
			apiBackend, err := mergeBackendBodies(definedBackends, apiBlock)
			if err != nil {
				return nil, err
			}

			parentBackend := mergeRight(apiBackend, serverBackend)
			err = refineEndpoints(definedBackends, parentBackend, apiBlock.Endpoints)
			if err != nil {
				return nil, err
			}
		}

		// standalone endpoints
		err := refineEndpoints(definedBackends, serverBackend, serverConfig.Endpoints)
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

// mergeBackendBodies appends the left side object with newly defined attributes or overrides already defined ones.
func mergeBackendBodies(definedBackends Backends, inline config.Inline) (hcl.Body, error) {
	reference, err := getReference(definedBackends, inline)
	if err != nil {
		return nil, err
	}

	content, _, diags := inline.HCLBody().PartialContent(inline.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	if content == nil {
		if reference != nil {
			return reference, nil
		}
		return nil, fmt.Errorf("configuration error: missing backend reference or inline definition")
	}

	// Apply current attributes to the referenced body.
	if len(content.Attributes) > 0 && reference != nil {
		reference = MergeBodies([]hcl.Body{reference, body.New(&hcl.BodyContent{
			Attributes:       content.Attributes,
			MissingItemRange: content.MissingItemRange,
		})})
	}

	var backendBlock *hcl.Block
	if backends := content.Blocks.OfType(backend); len(backends) > 0 {
		backendBlock = backends[0]
	} else {
		return reference, nil
	}

	// Case: `backend {}`, anonymous backend.
	if len(backendBlock.Labels) == 0 {
		return backendBlock.Body, nil
	}

	// Case: `backend "reference" {}`, referenced backend.
	refOverride, err := definedBackends.WithName(backendBlock.Labels[0])
	if err != nil {
		err.(hcl.Diagnostics)[0].Subject = &backendBlock.DefRange

		// Case: referenced backend is not defined in definitions.
		return nil, err
	}

	// link backend block name (label) to attribute 'name'
	if syntaxBody, ok := backendBlock.Body.(*hclsyntax.Body); ok {
		if refBody, ok := refOverride.(*hclsyntax.Body); ok {
			syntaxBody.Attributes[backendLabel] = refBody.Attributes[backendLabel]
		}
	}

	return MergeBodies([]hcl.Body{refOverride, backendBlock.Body}), nil
}

// mergeRight merges the right over the left one if the
// name label is the same, otherwise returns the right one.
func mergeRight(left hcl.Body, right hcl.Body) hcl.Body {
	if right != nil && left != nil {
		leftAttrs, _ := left.JustAttributes()
		leftLabel, ok := leftAttrs[backendLabel]
		if ok {
			rightAttrs, _ := right.JustAttributes()
			rightLabel, exist := rightAttrs[backendLabel]
			if exist && leftLabel == rightLabel {
				return MergeBodies([]hcl.Body{left, right})
			}
		}
	}
	return right
}

// getReference tries to fetch a backend from `definitions`
// block by a reference name, e.g. `backend = "name"`.
func getReference(definedBackends Backends, inline config.Inline) (hcl.Body, error) {
	reference, err := definedBackends.WithName(inline.Reference())
	if err != nil {
		// Backend reference is given, but not defined in definitions.
		r := inline.HCLBody().MissingItemRange()
		err.(hcl.Diagnostics)[0].Subject = &r
	}

	return reference, err
}

func refineEndpoints(definedBackends Backends, parentBackend hcl.Body, endpoints config.Endpoints) error {
	for _, endpoint := range endpoints {
		for i, proxyConfig := range endpoint.Proxies {
			bend, err := newBackend(definedBackends, parentBackend, proxyConfig)
			if err != nil {
				return err
			}
			endpoint.Proxies[i].Backend = bend
		}

		for i, reqConfig := range endpoint.Requests {
			bend, err := newBackend(definedBackends, parentBackend, reqConfig)
			if err != nil {
				return err
			}
			endpoint.Requests[i].Backend = bend
		}
	}

	return nil
}

func newBackend(definedBackends Backends, parentBackend hcl.Body, inlineConfig config.Inline) (hcl.Body, error) {
	bend, err := mergeBackendBodies(definedBackends, inlineConfig)
	if err != nil {
		return nil, err
	}

	bend = mergeRight(parentBackend, bend)
	if err = validateOrigin(bend); err != nil {
		r := inlineConfig.HCLBody().MissingItemRange()
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &r,
			Summary: err.Error(),
		}}
	}

	return bend, nil
}

// validateOrigin checks at least for an origin attribute definition.
func validateOrigin(merged hcl.Body) error {
	if merged == nil {
		return fmt.Errorf("missing backend reference or definition")
	}

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
