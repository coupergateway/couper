package test

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
)

func NewRemainContext(name, value string) hcl.Body {
	expr := hcltest.MockExprLiteral(cty.StringVal(value))
	return configload.MergeBodies([]hcl.Body{hcltest.MockBody(&hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
		name: {Name: name, Expr: expr},
	}})})
}
