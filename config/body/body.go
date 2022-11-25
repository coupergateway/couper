package body

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func NewHCLSyntaxBodyWithAttr(name string, value cty.Value, rng hcl.Range) *hclsyntax.Body {
	return &hclsyntax.Body{
		Attributes: hclsyntax.Attributes{
			name: {
				Name: name,
				Expr: &hclsyntax.LiteralValueExpr{Val: value, SrcRange: rng},
			},
		},
	}
}

func NewHCLSyntaxBodyWithStringAttr(name, value string) *hclsyntax.Body {
	return NewHCLSyntaxBodyWithAttr(name, cty.StringVal(value), hcl.Range{})
}

func MergeBodies(dest, src *hclsyntax.Body, replace bool) *hclsyntax.Body {
	if src == dest {
		return dest
	}
	for k, v := range src.Attributes {
		if _, set := dest.Attributes[k]; replace || !set {
			dest.Attributes[k] = v
		}
	}
	for _, bl := range src.Blocks {
		dest.Blocks = append(dest.Blocks, bl)
	}
	return dest
}

func BlocksOfType(body *hclsyntax.Body, blockType string) []*hclsyntax.Block {
	var blocks []*hclsyntax.Block
	for _, bl := range body.Blocks {
		if bl.Type == blockType {
			blocks = append(blocks, bl)
		}
	}
	return blocks
}

func RenameAttribute(body *hclsyntax.Body, old, new string) {
	if attr, ok := body.Attributes[old]; ok {
		attr.Name = new
		body.Attributes[new] = attr
		delete(body.Attributes, old)
	}
}
