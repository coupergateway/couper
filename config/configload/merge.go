package configload

import (
	"fmt"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

func errorUniqueLabels(block *hcl.Block) error {
	return hcl.Diagnostics{
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("All %s blocks must have unique labels.", block.Type),
			Subject:  &block.DefRange,
		},
	}
}

func mergeServers(bodies []hcl.Body) (hcl.Blocks, error) {
	type (
		namedBlocks map[string]*hcl.Block
		api         struct {
			labels       []string
			attributes   hcl.Attributes
			blocks       namedBlocks
			endpoints    namedBlocks
			errorHandler namedBlocks
		}
		namedAPIs map[string]*api
		server    struct {
			labels     []string
			attributes hcl.Attributes
			blocks     namedBlocks
			endpoints  namedBlocks
			apis       namedAPIs
		}
		servers map[string]*server
	)

	/*
		server[<key>] = {
			attributes       = hcl.Attributes
			blocks[<name>]   = hcl.Block (cors, files, spa)
			endpoints[<key>] = hcl.Block
			apis[<key>]      = {
				attributes           = hcl.Attributes
				blocks[<name>]       = hcl.Block (cors)
				endpoints[<key>]     = hcl.Block
				error_handler[<key>] = hcl.Block
			}
		}
	*/
	results := make(servers)

	for _, body := range bodies {
		uniqueServerLabels := make(map[string]struct{})

		for _, outerBlock := range bodyToContent(body).Blocks.OfType("server") {
			var serverKey string

			if len(outerBlock.Labels) > 0 {
				serverKey = outerBlock.Labels[0]
			}

			if len(bodies) > 1 {
				if _, ok := uniqueServerLabels[serverKey]; ok {
					return nil, errorUniqueLabels(outerBlock)
				}

				uniqueServerLabels[serverKey] = struct{}{}
			}

			if results[serverKey] == nil {
				results[serverKey] = &server{
					labels:     outerBlock.Labels,
					attributes: make(hcl.Attributes),
					blocks:     make(namedBlocks),
					endpoints:  make(namedBlocks),
					apis:       make(namedAPIs),
				}
			}

			serverContent := bodyToContent(outerBlock.Body)

			for name, attr := range serverContent.Attributes {
				results[serverKey].attributes[name] = attr
			}

			for _, block := range serverContent.Blocks {
				uniqueAPILabels := make(map[string]struct{})

				if block.Type == "endpoint" {
					if len(block.Labels) == 0 {
						return nil, errorUniqueLabels(block)
					}

					results[serverKey].endpoints[block.Labels[0]] = block
				} else if block.Type == "api" {
					var apiKey string

					if len(block.Labels) > 0 {
						apiKey = block.Labels[0]
					}

					if len(bodies) > 1 {
						if _, ok := uniqueAPILabels[apiKey]; ok {
							return nil, errorUniqueLabels(block)
						}

						uniqueAPILabels[apiKey] = struct{}{}
					}

					if results[serverKey].apis[apiKey] == nil {
						results[serverKey].apis[apiKey] = &api{
							labels:       block.Labels,
							attributes:   make(hcl.Attributes),
							blocks:       make(namedBlocks),
							endpoints:    make(namedBlocks),
							errorHandler: make(namedBlocks),
						}
					}

					apiContent := bodyToContent(block.Body)

					for name, attr := range apiContent.Attributes {
						results[serverKey].apis[apiKey].attributes[name] = attr
					}

					for _, subBlock := range apiContent.Blocks {
						if subBlock.Type == "endpoint" {
							if len(subBlock.Labels) == 0 {
								return nil, errorUniqueLabels(subBlock)
							}

							results[serverKey].apis[apiKey].endpoints[subBlock.Labels[0]] = subBlock
						} else if subBlock.Type == "error_handler" {
							var ehKey string

							if len(subBlock.Labels) > 0 {
								ehKey = subBlock.Labels[0]
							}

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

	var mergedServers hcl.Blocks

	for _, server := range results {
		var serverBlocks hcl.Blocks

		for _, b := range server.blocks {
			serverBlocks = append(serverBlocks, b)
		}

		for _, b := range server.endpoints {
			serverBlocks = append(serverBlocks, b)
		}

		for _, api := range server.apis {
			var apiBlocks hcl.Blocks

			for _, b := range api.blocks {
				apiBlocks = append(apiBlocks, b)
			}

			for _, b := range api.endpoints {
				apiBlocks = append(apiBlocks, b)
			}

			for _, b := range api.errorHandler {
				apiBlocks = append(apiBlocks, b)
			}

			mergedAPI := &hcl.Block{
				Type:   "api",
				Labels: api.labels,
				Body: hclbody.New(
					&hcl.BodyContent{
						Attributes: api.attributes,
						Blocks:     apiBlocks,
					},
				),
			}

			serverBlocks = append(serverBlocks, mergedAPI)
		}

		mergedServer := &hcl.Block{
			Type:   "server",
			Labels: server.labels,
			Body: hclbody.New(
				&hcl.BodyContent{
					Attributes: server.attributes,
					Blocks:     serverBlocks,
				},
			),
		}

		mergedServers = append(mergedServers, mergedServer)
	}

	return mergedServers, nil
}

func mergeDefinitions(bodies []hcl.Body) (*hcl.Block, error) {
	schema, _ := gohcl.ImpliedBodySchema(config.Definitions{})

	type data map[string]*hcl.Block
	type list map[string]data

	definitions := make(list)

	for _, body := range bodies {
		outerContent, err := contentByType("definitions", body)
		if err != nil {
			return nil, err
		}

		for _, outerBlock := range outerContent.Blocks {
			innerContent, diags := outerBlock.Body.Content(schema)
			if diags != nil {
				return nil, diags
			}

			for _, innerBlock := range innerContent.Blocks {
				if definitions[innerBlock.Type] == nil {
					definitions[innerBlock.Type] = make(data)
				}

				if len(innerBlock.Labels) == 0 {
					return nil, hcl.Diagnostics{
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  fmt.Sprintf("All %s blocks must have a label", innerBlock.Type),
							Subject:  &innerBlock.DefRange,
						},
					}
				}

				definitions[innerBlock.Type][innerBlock.Labels[0]] = innerBlock
			}
		}
	}

	var blocks []*hcl.Block

	for _, labels := range definitions {
		for _, block := range labels {
			blocks = append(blocks, block)
		}
	}

	return &hcl.Block{
		Type: "definitions",
		Body: hclbody.New(
			&hcl.BodyContent{
				Blocks: blocks,
			},
		),
	}, nil
}

func mergeAttributes(blockName string, bodies []hcl.Body) (*hcl.Block, error) {
	attrs := make(hcl.Attributes)

	for _, body := range bodies {
		content, err := contentByType(blockName, body)
		if err != nil {
			return nil, err
		}

		for _, block := range content.Blocks.OfType(blockName) {
			list, _ := block.Body.JustAttributes()

			for name, attr := range list {
				attrs[name] = attr
			}
		}
	}

	return &hcl.Block{
		Type: blockName,
		Body: hclbody.New(
			&hcl.BodyContent{
				Attributes: attrs,
			},
		),
	}, nil
}
