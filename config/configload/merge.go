package configload

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/avenga/couper/eval"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const (
	errMultipleBackends = "Multiple definitions of backend are not allowed."
	errUniqueLabels     = "All %s blocks must have unique labels."
)

func createError(msg string, block *hclsyntax.Block) error {
	defRange := block.DefRange()

	return hcl.Diagnostics{
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf(msg, block.Type),
			Subject:  &defRange,
		},
	}
}

// absPath replaces the given attribute path expression with an absolute path
// related to its filename if not already an absolute one.
func absPath(attr *hclsyntax.Attribute) hclsyntax.Expression {
	value, diags := attr.Expr.Value(envContext)
	if diags.HasErrors() || strings.Index(value.AsString(), "/") == 0 {
		return attr.Expr // Return unchanged in error cases and for absolute path values.
	}

	return &hclsyntax.LiteralValueExpr{
		Val: cty.StringVal(
			path.Join(filepath.Dir(attr.SrcRange.Filename), value.AsString()),
		),
		SrcRange: attr.SrcRange,
	}
}

func absBackendBlock(backendBlock *hclsyntax.Block) {
	for _, block := range backendBlock.Body.Blocks {
		if block.Type == "openapi" {
			if attr, ok := block.Body.Attributes["file"]; ok {
				block.Body.Attributes["file"].Expr = absPath(attr)
			}
		} else if block.Type == oauth2 {
			for _, innerBlock := range block.Body.Blocks {
				if innerBlock.Type == backend {
					absBackendBlock(innerBlock) // Recursive call
				}
			}
		}
	}
}

func absInBackends(block *hclsyntax.Block) error {
	for _, subBlock := range block.Body.Blocks {
		if subBlock.Type == errorHandler {
			return absInBackends(subBlock)
		}

		if subBlock.Type == proxy || subBlock.Type == request {
			var backends int

			for _, subSubBlock := range subBlock.Body.Blocks {
				fmt.Printf(">>> %#v\n", subSubBlock.Type)
				if subSubBlock.Type == backend {
					absBackendBlock(subSubBlock) // Backend block inside a proxy or request block

					backends++
				}
			}

			if _, ok := subBlock.Body.Attributes[backend]; ok {
				backends++
			}

			if backends > 1 {
				return createError(errMultipleBackends, block)
			}
		}
	}

	return nil
}

// newErrorHandlerKey returns a merge key based on a possible mixed error-kind format.
// "label1" and/or "label2 label3" results in "label1 label2 label3".
func newErrorHandlerKey(block *hclsyntax.Block) (key string) {
	if len(block.Labels) == 0 {
		return key
	}

	var sorted []string
	for _, l := range block.Labels {
		sorted = append(sorted, strings.Split(l, errorHandlerLabelSep)...)
	}
	sort.Strings(sorted)

	return strings.Join(sorted, errorHandlerLabelSep)
}

func mergeServers(bodies []*hclsyntax.Body) (hclsyntax.Blocks, error) {
	type (
		namedBlocks   map[string]*hclsyntax.Block
		apiDefinition struct {
			labels       []string
			attributes   hclsyntax.Attributes
			blocks       namedBlocks
			endpoints    namedBlocks
			errorHandler namedBlocks
		}
		namedAPIs        map[string]*apiDefinition
		serverDefinition struct {
			labels     []string
			attributes hclsyntax.Attributes
			blocks     namedBlocks
			endpoints  namedBlocks
			apis       namedAPIs
		}
		servers map[string]*serverDefinition
	)

	/*
		serverDefinition[<key>] = {
			attributes       = hclsyntax.Attributes
			blocks[<name>]   = hclsyntax.Block (cors, files, spa)
			endpoints[<key>] = hclsyntax.Block
			apis[<key>]      = {
				attributes           = hclsyntax.Attributes
				blocks[<name>]       = hclsyntax.Block (cors)
				endpoints[<key>]     = hclsyntax.Block
				error_handler[<key>] = hclsyntax.Block
			}
		}
	*/

	results := make(servers)

	for _, body := range bodies {
		uniqueServerLabels := make(map[string]struct{})

		for _, outerBlock := range body.Blocks {
			if outerBlock.Type != server {
				continue
			}

			var serverKey string

			if len(outerBlock.Labels) > 0 {
				serverKey = outerBlock.Labels[0]
			}

			if len(bodies) > 1 {
				if _, ok := uniqueServerLabels[serverKey]; ok {
					return nil, createError(errUniqueLabels, outerBlock)
				}

				uniqueServerLabels[serverKey] = struct{}{}
			} else {
				// Create unique key for multiple server blocks inside a single config file.
				serverKey += fmt.Sprintf("|%p", &serverKey)
			}

			if results[serverKey] == nil {
				results[serverKey] = &serverDefinition{
					labels:     outerBlock.Labels,
					attributes: make(hclsyntax.Attributes),
					blocks:     make(namedBlocks),
					endpoints:  make(namedBlocks),
					apis:       make(namedAPIs),
				}
			}

			for name, attr := range outerBlock.Body.Attributes {
				results[serverKey].attributes[name] = attr
			}

			if attr, ok := results[serverKey].attributes["error_file"]; ok {
				results[serverKey].attributes["error_file"].Expr = absPath(attr)
			}

			for _, block := range outerBlock.Body.Blocks {
				uniqueAPILabels := make(map[string]struct{})

				// TODO: Do we need this IF around the FOR?
				if block.Type == "files" || block.Type == "spa" || block.Type == api || block.Type == endpoint {
					for _, name := range []string{"error_file", "document_root"} {
						if attr, ok := block.Body.Attributes[name]; ok {
							block.Body.Attributes[name].Expr = absPath(attr)
						}
					}
				}

				if block.Type == api || block.Type == endpoint {
					for _, innerBlock := range block.Body.Blocks {
						if innerBlock.Type == errorHandler {
							if attr, ok := innerBlock.Body.Attributes["error_file"]; ok {
								innerBlock.Body.Attributes["error_file"].Expr = absPath(attr)
							}
						} else if innerBlock.Type == endpoint {
							for _, innerInnerBlock := range innerBlock.Body.Blocks {
								if innerInnerBlock.Type == backend {
									absBackendBlock(innerInnerBlock) // Backend block inside a endpoint block in an api block
								}
							}
						}
					}
				}

				if block.Type == endpoint {
					if err := absInBackends(block); err != nil { // Backend block inside a free endpoint block
						return nil, err
					}

					if len(block.Labels) == 0 {
						return nil, createError(errUniqueLabels, block)
					}

					results[serverKey].endpoints[block.Labels[0]] = block
				} else if block.Type == api {
					var apiKey string

					if len(block.Labels) > 0 {
						apiKey = block.Labels[0]
					}

					if len(bodies) > 1 {
						if _, ok := uniqueAPILabels[apiKey]; ok {
							return nil, createError(errUniqueLabels, block)
						}

						uniqueAPILabels[apiKey] = struct{}{}
					} else {
						// Create unique key for multiple api blocks inside a single config file.
						apiKey += fmt.Sprintf("|%p", &apiKey)
					}

					if results[serverKey].apis[apiKey] == nil {
						results[serverKey].apis[apiKey] = &apiDefinition{
							labels:       block.Labels,
							attributes:   make(hclsyntax.Attributes),
							blocks:       make(namedBlocks),
							endpoints:    make(namedBlocks),
							errorHandler: make(namedBlocks),
						}
					}

					for name, attr := range block.Body.Attributes {
						results[serverKey].apis[apiKey].attributes[name] = attr
					}

					for _, subBlock := range block.Body.Blocks {
						if subBlock.Type == endpoint {
							if err := absInBackends(subBlock); err != nil {
								return nil, err
							}

							if len(subBlock.Labels) == 0 {
								return nil, createError(errUniqueLabels, subBlock)
							}

							results[serverKey].apis[apiKey].endpoints[subBlock.Labels[0]] = subBlock
						} else if subBlock.Type == errorHandler {
							if err := absInBackends(subBlock); err != nil {
								return nil, err
							}

							ehKey := newErrorHandlerKey(subBlock)

							results[serverKey].apis[apiKey].errorHandler[ehKey] = subBlock
						} else {
							results[serverKey].apis[apiKey].blocks[subBlock.Type] = subBlock
						}
					}
				} else {
					results[serverKey].blocks[block.Type] = block
				}
			}
		}
	}

	var mergedServers hclsyntax.Blocks

	for _, serverBlock := range results {
		var serverBlocks hclsyntax.Blocks

		for _, b := range serverBlock.blocks {
			serverBlocks = append(serverBlocks, b)
		}

		for _, b := range serverBlock.endpoints {
			serverBlocks = append(serverBlocks, b)
		}

		for _, apiBlock := range serverBlock.apis {
			var apiBlocks hclsyntax.Blocks

			for _, b := range apiBlock.blocks {
				apiBlocks = append(apiBlocks, b)
			}

			for _, b := range apiBlock.endpoints {
				apiBlocks = append(apiBlocks, b)
			}

			for _, b := range apiBlock.errorHandler {
				apiBlocks = append(apiBlocks, b)
			}

			mergedAPI := &hclsyntax.Block{
				Type:   api,
				Labels: apiBlock.labels,
				Body: &hclsyntax.Body{
					Attributes: apiBlock.attributes,
					Blocks:     apiBlocks,
				},
			}

			serverBlocks = append(serverBlocks, mergedAPI)
		}

		mergedServer := &hclsyntax.Block{
			Type:   server,
			Labels: serverBlock.labels,
			Body: &hclsyntax.Body{
				Attributes: serverBlock.attributes,
				Blocks:     serverBlocks,
			},
		}

		mergedServers = append(mergedServers, mergedServer)
	}

	return mergedServers, nil
}

func mergeDefinitions(bodies []*hclsyntax.Body) (*hclsyntax.Block, error) {
	type data map[string]*hclsyntax.Block
	type list map[string]data

	definitionsBlock := make(list)

	for _, body := range bodies {
		for _, outerBlock := range body.Blocks {
			if outerBlock.Type == definitions {
				for _, innerBlock := range outerBlock.Body.Blocks {
					if definitionsBlock[innerBlock.Type] == nil {
						definitionsBlock[innerBlock.Type] = make(data)
					}

					if len(innerBlock.Labels) == 0 {
						return nil, createError(errUniqueLabels, innerBlock)
					}

					definitionsBlock[innerBlock.Type][innerBlock.Labels[0]] = innerBlock

					// TODO: Do we need this IF around the FOR?
					if innerBlock.Type == "basic_auth" || innerBlock.Type == "jwt" || innerBlock.Type == "jwt_signing_profile" || innerBlock.Type == "saml" {
						for _, name := range []string{"htpasswd_file", "key_file", "signing_key_file", "idp_metadata_file"} {
							if attr, ok := innerBlock.Body.Attributes[name]; ok {
								innerBlock.Body.Attributes[name].Expr = absPath(attr)
							}
						}
					}

					var backends int

					for _, block := range innerBlock.Body.Blocks {
						if block.Type == errorHandler {
							if attr, ok := block.Body.Attributes["error_file"]; ok {
								block.Body.Attributes["error_file"].Expr = absPath(attr)
							}

							if err := absInBackends(block); err != nil {
								return nil, err
							}
						} else if block.Type == backend {
							absBackendBlock(block) // Backend block inside an AC block

							backends++
						}
					}

					if _, ok := innerBlock.Body.Attributes[backend]; ok {
						backends++
					}

					if backends > 1 {
						return nil, createError(errMultipleBackends, innerBlock)
					}

					if innerBlock.Type == backend {
						absBackendBlock(innerBlock) // Backend block inside the definitions block
					}
				}
			}
		}
	}

	var blocks []*hclsyntax.Block

	for _, labels := range definitionsBlock {
		for _, block := range labels {
			blocks = append(blocks, block)
		}
	}

	return &hclsyntax.Block{
		Type: definitions,
		Body: &hclsyntax.Body{
			Blocks: blocks,
		},
	}, nil
}

func mergeDefaults(bodies []*hclsyntax.Body) (*hclsyntax.Block, error) {
	attrs := make(hclsyntax.Attributes)
	envVars := make(map[string]cty.Value)

	for _, body := range bodies {
		for _, block := range body.Blocks {
			if block.Type == defaults {
				for name, attr := range block.Body.Attributes {
					if name == "environment_variables" {
						v, err := eval.Value(nil, attr.Expr)
						if err != nil {
							return nil, err
						}

						for name, value := range v.AsValueMap() {
							if value.Type() != cty.String {
								return nil, fmt.Errorf("value in 'environment_variables' is not a string")
							}

							envVars[name] = value
						}
					} else {
						attrs[name] = attr // Currently not used
					}
				}
			}
		}
	}

	if len(envVars) > 0 {
		attrs["environment_variables"] = &hclsyntax.Attribute{
			Name: "environment_variables",
			Expr: &hclsyntax.LiteralValueExpr{
				Val: cty.MapVal(envVars),
			},
		}
	}

	return &hclsyntax.Block{
		Type: defaults,
		Body: &hclsyntax.Body{
			Attributes: attrs,
		},
	}, nil
}

func mergeSettings(bodies []*hclsyntax.Body) *hclsyntax.Block {
	attrs := make(hclsyntax.Attributes)

	for _, body := range bodies {
		for _, block := range body.Blocks {
			if block.Type == settings {
				for name, attr := range block.Body.Attributes {
					if name == "ca_file" {
						block.Body.Attributes[name].Expr = absPath(attr)
					}

					attrs[name] = attr
				}
			}
		}
	}

	return &hclsyntax.Block{
		Type: settings,
		Body: &hclsyntax.Body{
			Attributes: attrs,
		},
	}
}
