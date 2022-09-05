package test

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func NewRemainContext(name, value string) *hclsyntax.Body {
	expr := &hclsyntax.LiteralValueExpr{Val: cty.StringVal(value)}
	return &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
		name: {Name: name, Expr: expr},
	}}
}
