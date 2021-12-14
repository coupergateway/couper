package configload

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
)

// addSequenceDeps collects possible dependencies from variables.
func addSequenceDeps(names map[string]struct{}, endpoint *config.Endpoint) {
	var items []config.SequenceItem
	for _, conf := range endpoint.Proxies {
		items = append(items, conf)
	}
	for _, conf := range endpoint.Requests {
		items = append(items, conf)
	}

	for _, seqItem := range items {
		allExpressions := collectExpressions(seqItem.HCLBody())
		for _, expr := range allExpressions {
			for _, traversal := range expr.Variables() {
				if traversal.RootName() != "backend_responses" || len(traversal) < 2 {
					continue
				}

				// do we have a ref ?
				for _, t := range traversal[1:] {
					tr, ok := t.(hcl.TraverseAttr)
					if !ok {
						continue
					}

					_, ok = names[tr.Name]
					if !ok {
						continue
					}

					for _, i := range items {
						if i.GetName() != tr.Name || i == seqItem {
							continue
						}
						seqItem.Add(i)
						break
					}
				}
			}
		}
	}
}

func collectExpressions(bodies ...hcl.Body) []hcl.Expression {
	allExpressions := make([]hcl.Expression, 0)

	for _, b := range bodies {
		switch sb := b.(type) {
		case *hclsyntax.Body:
			for _, attr := range sb.Attributes {
				allExpressions = append(allExpressions, attr.Expr)
			}

			for _, block := range sb.Blocks {
				allExpressions = append(allExpressions, collectExpressions(block.Body)...)
			}
		case *body.Body:
			content, _, _ := sb.PartialContent(nil)
			for _, attr := range content.Attributes {
				allExpressions = append(allExpressions, attr.Expr)
			}
			for _, block := range content.Blocks {
				allExpressions = append(allExpressions, collectExpressions(block.Body)...)
			}
		case body.MergedBodies:
			// top-level attrs
			for _, attrs := range sb.JustAllAttributes() {
				for _, attr := range attrs {
					allExpressions = append(allExpressions, attr.Expr)
				}
			}

			// nested block attrs
			for _, mb := range sb {
				allExpressions = append(allExpressions, collectExpressions(mb)...)
			}
		}
	}
	return allExpressions
}
