package body

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func CollectAttributes(bodies ...hcl.Body) []*hcl.Attribute {
	allAttributes := make([]*hcl.Attribute, 0)

	for _, b := range bodies {
		switch sb := b.(type) {
		case *hclsyntax.Body:
			for _, attr := range sb.Attributes {
				allAttributes = append(allAttributes, &hcl.Attribute{
					Name:      attr.Name,
					Expr:      attr.Expr,
					Range:     attr.SrcRange,
					NameRange: attr.NameRange,
				})
			}

			for _, block := range sb.Blocks {
				allAttributes = append(allAttributes, CollectAttributes(block.Body)...)
			}
		case *Body:
			content, _, _ := sb.PartialContent(nil)
			for _, attr := range content.Attributes {
				allAttributes = append(allAttributes, attr)
			}
			for _, block := range content.Blocks {
				allAttributes = append(allAttributes, CollectAttributes(block.Body)...)
			}
		case MergedBodies:
			// top-level attrs
			for _, attrs := range sb.JustAllAttributes() {
				for _, attr := range attrs {
					allAttributes = append(allAttributes, attr)
				}
			}

			// nested block attrs
			for _, mb := range sb {
				allAttributes = append(allAttributes, CollectAttributes(mb)...)
			}
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
		switch sb := b.(type) {
		case *hclsyntax.Body:

			for _, block := range sb.Blocks {
				nested := append(append([]string{}, block.Type), CollectBlockTypes(block.Body)...)
				addUniqueFn(nested...)
			}
		case *Body:
			content, _, _ := sb.PartialContent(nil)
			for _, block := range content.Blocks {
				nested := append(append([]string{}, block.Type), CollectBlockTypes(block.Body)...)
				addUniqueFn(nested...)
			}
		case MergedBodies:
			// nested block
			for _, mb := range sb {
				nested := append([]string{}, CollectBlockTypes(mb)...)
				addUniqueFn(nested...)
			}
		}
	}

	var result []string
	for u := range unique {
		result = append(result, u)
	}

	return result
}

func CollectExpressions(bodies ...hcl.Body) []hcl.Expression {
	allExpressions := make([]hcl.Expression, 0)
	for _, attr := range CollectAttributes(bodies...) {
		allExpressions = append(allExpressions, attr.Expr)
	}

	return allExpressions
}
