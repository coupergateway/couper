package body

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var _ hcl.Body = &body{}

type Attributes interface {
	JustAllAttributes() []hcl.Attributes
	JustAllAttributesWithName(string) []hcl.Attributes
}

type body struct {
	content *hcl.BodyContent
}

func NewBody(content *hcl.BodyContent) hcl.Body {
	return &body{content}
}

func NewHCLSyntaxBodyWithStringAttr(name string, value cty.Value, rng hcl.Range) *hclsyntax.Body {
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

func MergeBds(dest, src *hclsyntax.Body, replace bool) *hclsyntax.Body {
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

func (e *body) Content(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	return e.content, nil
}

func (e *body) PartialContent(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	return e.content, e, nil
}

func (e *body) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	attrs := hcl.Attributes{}
	for k, v := range e.content.Attributes {
		cv := *v
		attrs[k] = &cv
	}
	return attrs, nil
}

func (e *body) MissingItemRange() hcl.Range {
	return e.content.MissingItemRange
}
