package configload

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
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
		case mergedBodies:
			for _, attrs := range sb.JustAllAttributes() {
				for _, attr := range attrs {
					allExpressions = append(allExpressions, attr.Expr)
				}
			}
		}
	}
	return allExpressions
}
