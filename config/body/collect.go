package body

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func CollectAttributes(bodies ...hcl.Body) []*hclsyntax.Attribute {
	allAttributes := make([]*hclsyntax.Attribute, 0)

	for _, b := range bodies {
		sb, _ := b.(*hclsyntax.Body)
		for _, attr := range sb.Attributes {
			allAttributes = append(allAttributes, attr)
		}

		for _, block := range sb.Blocks {
			allAttributes = append(allAttributes, CollectAttributes(block.Body)...)
		}
	}

	return allAttributes
}

func CollectBlockTypes(bodies ...hcl.Body) []string {
	unique := make(map[string]struct{})

	addUniqueFn := func(types ...string) {
		for _, t := range types {
			if _, exist := unique[t]; exist {
				continue
			}
			unique[t] = struct{}{}
		}
	}

	for _, b := range bodies {
		sb, _ := b.(*hclsyntax.Body)
		for _, block := range sb.Blocks {
			nested := append(append([]string{}, block.Type), CollectBlockTypes(block.Body)...)
			addUniqueFn(nested...)
		}
	}

	var result []string
	for u := range unique {
		result = append(result, u)
	}

	return result
}

func CollectExpressions(bodies ...hcl.Body) []hclsyntax.Expression {
	allExpressions := make([]hclsyntax.Expression, 0)
	for _, attr := range CollectAttributes(bodies...) {
		allExpressions = append(allExpressions, attr.Expr)
	}

	return allExpressions
}
