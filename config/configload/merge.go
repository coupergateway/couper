package configload

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const (
	errMultipleBackends = "Multiple definitions of backend are not allowed in %s."
	errUniqueLabels     = "All %s blocks must have unique labels."
)

type namedBlocks map[string]*hclsyntax.Block

func mergeServers(bodies []*hclsyntax.Body, proxies map[string]*hclsyntax.Block) (hclsyntax.Blocks, error) {
	type (
		apiDefinition struct {
			labels          []string
			typeRange       hcl.Range
			labelRanges     []hcl.Range
			openBraceRange  hcl.Range
			closeBraceRange hcl.Range
			attributes      hclsyntax.Attributes
			blocks          namedBlocks
			endpoints       namedBlocks
			errorHandler    namedBlocks
		}
		spaDefinition struct {
			labels          []string
			typeRange       hcl.Range
			labelRanges     []hcl.Range
			openBraceRange  hcl.Range
			closeBraceRange hcl.Range
			attributes      hclsyntax.Attributes
			blocks          namedBlocks
		}
		filesDefinition struct {
			labels          []string
			typeRange       hcl.Range
			labelRanges     []hcl.Range
			openBraceRange  hcl.Range
			closeBraceRange hcl.Range
			attributes      hclsyntax.Attributes
			blocks          namedBlocks
		}

		tlsDefinition struct {
			*hclsyntax.Block
			typeBlocks      map[string][]*hclsyntax.Block
			preservedBlocks namedBlocks
		}

		namedAPIs  map[string]*apiDefinition
		namedSPAs  map[string]*spaDefinition
		namedFiles map[string]*filesDefinition

		serverDefinition struct {
			labels          []string
			typeRange       hcl.Range
			labelRanges     []hcl.Range
			openBraceRange  hcl.Range
			closeBraceRange hcl.Range
			attributes      hclsyntax.Attributes
			apis            namedAPIs
			blocks          namedBlocks
			endpoints       namedBlocks
			files           namedFiles
			spas            namedSPAs
			tls             *tlsDefinition
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
			spas[<key>]      = {
				attributes           = hclsyntax.Attributes
				blocks[<name>]       = hclsyntax.Block (cors)
			}
			files[<key>]      = {
				attributes           = hclsyntax.Attributes
				blocks[<name>]       = hclsyntax.Block (cors)
			}
			tls = {
				blocks[<name>]       = hclsyntax.Block (server_certificate|client_certificate)
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
					return nil, newMergeError(errUniqueLabels, outerBlock)
				}

				uniqueServerLabels[serverKey] = struct{}{}
			} else {
				// Create unique key for multiple server blocks inside a single config file.
				serverKey += fmt.Sprintf("|%p", &serverKey)
			}

			if results[serverKey] == nil {
				results[serverKey] = &serverDefinition{
					labels:          outerBlock.Labels,
					typeRange:       outerBlock.TypeRange,
					labelRanges:     outerBlock.LabelRanges,
					openBraceRange:  outerBlock.OpenBraceRange,
					closeBraceRange: outerBlock.CloseBraceRange,
					attributes:      make(hclsyntax.Attributes),
					blocks:          make(namedBlocks),
					endpoints:       make(namedBlocks),
					apis:            make(namedAPIs),
					spas:            make(namedSPAs),
					files:           make(namedFiles),
				}
			}

			for name, attr := range outerBlock.Body.Attributes {
				results[serverKey].attributes[name] = attr
			}

			for _, block := range outerBlock.Body.Blocks {
				uniqueAPILabels, uniqueSPALabels, uniqueFilesLabels := make(map[string]struct{}), make(map[string]struct{}), make(map[string]struct{})

				if block.Type == endpoint {
					if err := checkForMultipleBackends(block); err != nil { // Backend block inside a free endpoint block
						return nil, err
					}

					if len(block.Labels) == 0 {
						return nil, newMergeError(errUniqueLabels, block)
					}

					if err := addProxy(block, proxies); err != nil {
						return nil, err
					}

					results[serverKey].endpoints[block.Labels[0]] = block
				} else if block.Type == api {
					var apiKey string

					if len(block.Labels) > 0 {
						apiKey = block.Labels[0]
					}

					if len(bodies) > 1 {
						if _, ok := uniqueAPILabels[apiKey]; ok {
							return nil, newMergeError(errUniqueLabels, block)
						}

						uniqueAPILabels[apiKey] = struct{}{}
					} else {
						// Create unique key for multiple api blocks inside a single config file.
						apiKey += fmt.Sprintf("|%p", &apiKey)
					}

					if results[serverKey].apis[apiKey] == nil {
						results[serverKey].apis[apiKey] = &apiDefinition{
							labels:          block.Labels,
							typeRange:       block.TypeRange,
							labelRanges:     block.LabelRanges,
							openBraceRange:  block.OpenBraceRange,
							closeBraceRange: block.CloseBraceRange,
							attributes:      make(hclsyntax.Attributes),
							blocks:          make(namedBlocks),
							endpoints:       make(namedBlocks),
							errorHandler:    make(namedBlocks),
						}
					}

					for name, attr := range block.Body.Attributes {
						results[serverKey].apis[apiKey].attributes[name] = attr
					}

					for _, subBlock := range block.Body.Blocks {
						if subBlock.Type == endpoint {
							if err := checkForMultipleBackends(subBlock); err != nil {
								return nil, err
							}

							if len(subBlock.Labels) == 0 {
								return nil, newMergeError(errUniqueLabels, subBlock)
							}

							if err := addProxy(subBlock, proxies); err != nil {
								return nil, err
							}

							results[serverKey].apis[apiKey].endpoints[subBlock.Labels[0]] = subBlock
						} else if subBlock.Type == errorHandler {
							if err := checkForMultipleBackends(subBlock); err != nil {
								return nil, err
							}

							ehKey := newErrorHandlerKey(subBlock)

							results[serverKey].apis[apiKey].errorHandler[ehKey] = subBlock
						} else {
							results[serverKey].apis[apiKey].blocks[subBlock.Type] = subBlock
						}
					}
				} else if block.Type == spa {
					var spaKey string

					if len(block.Labels) > 0 {
						spaKey = block.Labels[0]
					}

					if len(bodies) > 1 {
						if _, ok := uniqueSPALabels[spaKey]; ok {
							return nil, newMergeError(errUniqueLabels, block)
						}

						uniqueSPALabels[spaKey] = struct{}{}
					} else {
						// Create unique key for multiple spa blocks inside a single config file.
						spaKey += fmt.Sprintf("|%p", &spaKey)
					}

					if results[serverKey].spas[spaKey] == nil {
						results[serverKey].spas[spaKey] = &spaDefinition{
							labels:          block.Labels,
							typeRange:       block.TypeRange,
							labelRanges:     block.LabelRanges,
							openBraceRange:  block.OpenBraceRange,
							closeBraceRange: block.CloseBraceRange,
							attributes:      make(hclsyntax.Attributes),
							blocks:          make(namedBlocks),
						}
					}

					for name, attr := range block.Body.Attributes {
						results[serverKey].spas[spaKey].attributes[name] = attr
					}

					for _, subBlock := range block.Body.Blocks {
						results[serverKey].spas[spaKey].blocks[subBlock.Type] = subBlock
					}
				} else if block.Type == files {
					var filesKey string

					if len(block.Labels) > 0 {
						filesKey = block.Labels[0]
					}

					if len(bodies) > 1 {
						if _, ok := uniqueFilesLabels[filesKey]; ok {
							return nil, newMergeError(errUniqueLabels, block)
						}

						uniqueFilesLabels[filesKey] = struct{}{}
					} else {
						// Create unique key for multiple files blocks inside a single config file.
						filesKey += fmt.Sprintf("|%p", &filesKey)
					}

					if results[serverKey].files[filesKey] == nil {
						results[serverKey].files[filesKey] = &filesDefinition{
							labels:          block.Labels,
							typeRange:       block.TypeRange,
							labelRanges:     block.LabelRanges,
							openBraceRange:  block.OpenBraceRange,
							closeBraceRange: block.CloseBraceRange,
							attributes:      make(hclsyntax.Attributes),
							blocks:          make(namedBlocks),
						}
					}

					for name, attr := range block.Body.Attributes {
						results[serverKey].files[filesKey].attributes[name] = attr
					}

					for _, subBlock := range block.Body.Blocks {
						results[serverKey].files[filesKey].blocks[subBlock.Type] = subBlock
					}
				} else if block.Type == tls {
					if results[serverKey].tls == nil {
						results[serverKey].tls = &tlsDefinition{
							Block:           block,
							preservedBlocks: namedBlocks{},
							typeBlocks:      map[string][]*hclsyntax.Block{},
						}
					}

					for name, attr := range block.Body.Attributes {
						results[serverKey].tls.Body.Attributes[name] = attr
					}

					thisTypeBlocks := map[string][]*hclsyntax.Block{}
					for _, subBlock := range block.Body.Blocks {
						blockKey := subBlock.Type
						if len(subBlock.Labels) > 0 {
							blockKey += "_" + subBlock.Labels[0]
							results[serverKey].tls.preservedBlocks[blockKey] = subBlock // override
						} else {
							thisTypeBlocks[subBlock.Type] = append(thisTypeBlocks[subBlock.Type], subBlock)
						}
					}

					for t, v := range thisTypeBlocks { // reset per file
						if len(v) > 0 {
							results[serverKey].tls.typeBlocks[t] = v
						}

					}

				} else {
					results[serverKey].blocks[block.Type] = block
				}
			}
		}
	}

	var mergedServers hclsyntax.Blocks

	for _, name := range getSortedMapKeys(results) {
		var serverBlocks hclsyntax.Blocks
		serverBlock := results[name]
		for _, blockName := range getSortedMapKeys(serverBlock.blocks) {
			serverBlocks = append(serverBlocks, serverBlock.blocks[blockName])
		}

		for _, blockName := range getSortedMapKeys(serverBlock.endpoints) {
			serverBlocks = append(serverBlocks, serverBlock.endpoints[blockName])
		}

		for _, apiBlock := range serverBlock.apis {
			var apiBlocks hclsyntax.Blocks

			for _, blockName := range getSortedMapKeys(apiBlock.blocks) {
				apiBlocks = append(apiBlocks, apiBlock.blocks[blockName])
			}

			for _, blockName := range getSortedMapKeys(apiBlock.endpoints) {
				apiBlocks = append(apiBlocks, apiBlock.endpoints[blockName])
			}

			for _, blockName := range getSortedMapKeys(apiBlock.errorHandler) {
				apiBlocks = append(apiBlocks, apiBlock.errorHandler[blockName])
			}

			mergedAPI := &hclsyntax.Block{
				Type:   api,
				Labels: apiBlock.labels,
				Body: &hclsyntax.Body{
					Attributes: apiBlock.attributes,
					Blocks:     apiBlocks,
				},
				TypeRange:       apiBlock.typeRange,
				LabelRanges:     apiBlock.labelRanges,
				OpenBraceRange:  apiBlock.openBraceRange,
				CloseBraceRange: apiBlock.closeBraceRange,
			}

			serverBlocks = append(serverBlocks, mergedAPI)
		}

		for _, blockName := range getSortedMapKeys(serverBlock.spas) {
			spaBlock := serverBlock.spas[blockName]
			var spaBlocks hclsyntax.Blocks

			for _, bn := range getSortedMapKeys(spaBlock.blocks) {
				spaBlocks = append(spaBlocks, spaBlock.blocks[bn])
			}

			mergedSPA := &hclsyntax.Block{
				Type:   spa,
				Labels: spaBlock.labels,
				Body: &hclsyntax.Body{
					Attributes: spaBlock.attributes,
					Blocks:     spaBlocks,
				},
				TypeRange:       spaBlock.typeRange,
				LabelRanges:     spaBlock.labelRanges,
				OpenBraceRange:  spaBlock.openBraceRange,
				CloseBraceRange: spaBlock.closeBraceRange,
			}

			serverBlocks = append(serverBlocks, mergedSPA)
		}

		for _, blockName := range getSortedMapKeys(serverBlock.files) {
			filesBlock := serverBlock.files[blockName]
			var filesBlocks hclsyntax.Blocks

			for _, bn := range getSortedMapKeys(filesBlock.blocks) {
				filesBlocks = append(filesBlocks, filesBlock.blocks[bn])
			}

			mergedFiles := &hclsyntax.Block{
				Type:   files,
				Labels: filesBlock.labels,
				Body: &hclsyntax.Body{
					Attributes: filesBlock.attributes,
					Blocks:     filesBlocks,
				},
				TypeRange:       filesBlock.typeRange,
				LabelRanges:     filesBlock.labelRanges,
				OpenBraceRange:  filesBlock.openBraceRange,
				CloseBraceRange: filesBlock.closeBraceRange,
			}

			serverBlocks = append(serverBlocks, mergedFiles)
		}

		if serverBlock.tls != nil {
			var tlsCertificateBlocks hclsyntax.Blocks
			for _, subType := range getSortedMapKeys(serverBlock.tls.typeBlocks) {
				tlsCertificateBlocks = append(tlsCertificateBlocks, serverBlock.tls.typeBlocks[subType]...)
			}
			for _, k := range getSortedMapKeys(serverBlock.tls.preservedBlocks) {
				tlsCertificateBlocks = append(tlsCertificateBlocks, serverBlock.tls.preservedBlocks[k])
			}
			serverBlocks = append(serverBlocks, &hclsyntax.Block{
				Type: tls,
				Body: &hclsyntax.Body{
					Attributes: serverBlock.tls.Body.Attributes,
					Blocks:     tlsCertificateBlocks,
				},
				TypeRange:       serverBlock.tls.TypeRange,
				LabelRanges:     serverBlock.tls.LabelRanges,
				OpenBraceRange:  serverBlock.tls.OpenBraceRange,
				CloseBraceRange: serverBlock.tls.CloseBraceRange,
			})
		}

		mergedServer := &hclsyntax.Block{
			Type:   server,
			Labels: serverBlock.labels,
			Body: &hclsyntax.Body{
				Attributes: serverBlock.attributes,
				Blocks:     serverBlocks,
			},
			TypeRange:       serverBlock.typeRange,
			LabelRanges:     serverBlock.labelRanges,
			OpenBraceRange:  serverBlock.openBraceRange,
			CloseBraceRange: serverBlock.closeBraceRange,
		}

		mergedServers = append(mergedServers, mergedServer)
	}

	return mergedServers, nil
}

func mergeDefinitions(bodies []*hclsyntax.Body) (*hclsyntax.Block, map[string]*hclsyntax.Block, error) {
	definitionsBlock := make(map[string]namedBlocks)
	proxiesList := make(namedBlocks)

	for _, body := range bodies {
		for _, outerBlock := range body.Blocks {
			if outerBlock.Type == definitions {
				for _, innerBlock := range outerBlock.Body.Blocks {
					if definitionsBlock[innerBlock.Type] == nil {
						definitionsBlock[innerBlock.Type] = make(namedBlocks)
					}

					if innerBlock.Type == backend {
						if err := checkForMultipleBackendsInBackend(innerBlock); err != nil {
							return nil, nil, err
						}
					}

					// Count the "backend" blocks and "backend" attributes to
					// forbid multiple backend definitions.

					var backends int

					for _, block := range innerBlock.Body.Blocks {
						if block.Type == errorHandler {
							if err := checkForMultipleBackends(block); err != nil {
								return nil, nil, err
							}
						} else if block.Type == backend {
							backends++
							if err := checkForMultipleBackendsInBackend(block); err != nil {
								return nil, nil, err
							}
						}
					}

					if _, ok := innerBlock.Body.Attributes[backend]; ok {
						backends++
					}

					if backends > 1 {
						return nil, nil, newMergeError(errMultipleBackends, innerBlock)
					}

					if innerBlock.Type != proxy {
						definitionsBlock[innerBlock.Type][innerBlock.Labels[0]] = innerBlock
					} else {
						label := innerBlock.Labels[0]

						if attr, ok := innerBlock.Body.Attributes["name"]; ok {
							name, err := attrStringValue(attr)
							if err != nil {
								return nil, nil, err
							}

							innerBlock.Labels[0] = name

							delete(innerBlock.Body.Attributes, "name")
						} else {
							innerBlock.Labels[0] = config.DefaultNameLabel
						}

						proxiesList[label] = innerBlock
					}
				}
			}
		}
	}

	var blocks []*hclsyntax.Block

	for _, name := range getSortedMapKeys(definitionsBlock) {
		for _, label := range getSortedMapKeys(definitionsBlock[name]) {
			blocks = append(blocks, definitionsBlock[name][label])
		}
	}

	return &hclsyntax.Block{
		Type: definitions,
		Body: &hclsyntax.Body{
			Blocks: blocks,
		},
	}, proxiesList, nil
}

func mergeDefaults(bodies []*hclsyntax.Body) (*hclsyntax.Block, error) {
	attrs := make(hclsyntax.Attributes)
	envVars := make(map[string]hclsyntax.Expression)

	for _, body := range bodies {
		for _, block := range body.Blocks {
			if block.Type != defaults {
				continue
			}

			for name, attr := range block.Body.Attributes {
				if name != environmentVars {
					attrs[name] = attr // Currently not used
					continue
				}

				expObj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
				if !ok {
					r := attr.Expr.Range()
					return nil, newDiagErr(&r, fmt.Sprintf("%s must be object type", environmentVars))
				}

				for _, item := range expObj.Items {
					k := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr)
					r := item.KeyExpr.Range()
					var keyName string
					switch exp := k.Wrapped.(type) {
					case *hclsyntax.ScopeTraversalExpr:
						if len(exp.Traversal) > 1 {
							return nil, newDiagErr(&r, "unsupported key scope traversal expression")
						}
						keyName = exp.Traversal.RootName()
					case *hclsyntax.TemplateExpr:
						if !exp.IsStringLiteral() {
							return nil, newDiagErr(&r, "unsupported key template expression")
						}
						v, _ := exp.Value(nil)
						keyName = v.AsString()
					default:
						r := item.KeyExpr.Range()
						return nil, newDiagErr(&r, "unsupported key expression")
					}

					envVars[keyName] = item.ValueExpr
				}
			}
		}
	}

	if len(envVars) > 0 {
		items := make([]hclsyntax.ObjectConsItem, 0)
		for k, v := range envVars {
			items = append(items, hclsyntax.ObjectConsItem{
				KeyExpr: &hclsyntax.ObjectConsKeyExpr{
					Wrapped: &hclsyntax.ScopeTraversalExpr{
						Traversal: hcl.Traversal{hcl.TraverseRoot{Name: k}},
					},
					ForceNonLiteral: false,
				},
				ValueExpr: v,
			})
		}

		attrs[environmentVars] = &hclsyntax.Attribute{
			Name: environmentVars,
			Expr: &hclsyntax.ObjectConsExpr{
				Items: items,
			},
		}
	}

	defaultsBlock := &hclsyntax.Block{
		Type: defaults,
		Body: &hclsyntax.Body{
			Attributes: attrs,
		},
	}
	return defaultsBlock, nil
}

func mergeSettings(bodies []*hclsyntax.Body) *hclsyntax.Block {
	attrs := make(hclsyntax.Attributes)

	for _, body := range bodies {
		for _, block := range body.Blocks {
			if block.Type == settings {
				for name, attr := range block.Body.Attributes {
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

func newMergeError(msg string, block *hclsyntax.Block) error {
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

// checkForMultipleBackends searches for "backend" blocks inside a proxy or request block to
// count the "backend"
// blocks and "backend" attributes to forbid multiple backend definitions.
func checkForMultipleBackends(block *hclsyntax.Block) error {
	for _, subBlock := range block.Body.Blocks {
		if subBlock.Type == errorHandler {
			return checkForMultipleBackends(subBlock) // Recursive call
		}

		if subBlock.Type != proxy && subBlock.Type != request {
			continue
		}

		var backends int

		for _, subSubBlock := range subBlock.Body.Blocks {
			if subSubBlock.Type == backend {
				backends++
				if err := checkForMultipleBackendsInBackend(subSubBlock); err != nil {
					return err
				}
			}
		}

		if _, ok := subBlock.Body.Attributes[backend]; ok {
			backends++
		}

		if backends > 1 {
			return newMergeError(errMultipleBackends, subBlock)
		}
	}

	return nil
}

func checkForMultipleBackendsInBackend(block *hclsyntax.Block) error {
	for _, subBlock := range block.Body.Blocks {
		if subBlock.Type != oauth2 && subBlock.Type != tokenRequest {
			continue
		}

		var backends int

		for _, subSubBlock := range subBlock.Body.Blocks {
			if subSubBlock.Type == backend {
				backends++
				if err := checkForMultipleBackendsInBackend(subSubBlock); err != nil { // Recursive call
					return err
				}
			}
		}

		if _, ok := subBlock.Body.Attributes[backend]; ok {
			backends++
		}

		if backends > 1 {
			return newMergeError(errMultipleBackends, subBlock)
		}
	}

	return nil
}

func addProxy(block *hclsyntax.Block, proxies map[string]*hclsyntax.Block) error {
	if attr, ok := block.Body.Attributes[proxy]; ok {
		reference, err := attrStringValue(attr)
		if err != nil {
			return err
		}

		proxyBlock, ok := proxies[reference]
		if !ok {
			sr := attr.Expr.StartRange()

			return newDiagErr(&sr, "proxy reference is not defined")
		}

		delete(block.Body.Attributes, proxy)

		block.Body.Blocks = append(block.Body.Blocks, proxyBlock)
	}

	return nil
}

func attrStringValue(attr *hclsyntax.Attribute) (string, error) {
	v, err := eval.Value(nil, attr.Expr)
	if err != nil {
		return "", err
	}

	if v.Type() != cty.String {
		sr := attr.Expr.StartRange()
		return "", newDiagErr(&sr, fmt.Sprintf("%s must evaluate to string", attr.Name))
	}

	return v.AsString(), nil
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

func getSortedMapKeys[K string, V any](m map[K]V) []string {
	var result []string
	for k := range m {
		result = append(result, string(k))
	}
	sort.Strings(result)
	return result
}
