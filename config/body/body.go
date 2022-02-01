package body

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var _ hcl.Body = &Body{}

type Attributes interface {
	JustAllAttributes() []hcl.Attributes
	JustAllAttributesWithName(string) []hcl.Attributes
}

type Body struct {
	content *hcl.BodyContent
}

func New(content *hcl.BodyContent) hcl.Body {
	return &Body{content}
}

func NewContentWithAttrName(name, value string) *hcl.BodyContent {
	return &hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{
			name: {
				Name: name,
				Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(value)},
			},
		},
	}
}

func (e *Body) Content(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	return e.content, nil
}

func (e *Body) PartialContent(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	return e.content, e, nil
}

func (e *Body) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	attrs := hcl.Attributes{}
	for k, v := range e.content.Attributes {
		cv := *v
		attrs[k] = &cv
	}
	return attrs, nil
}

func (e *Body) MissingItemRange() hcl.Range {
	return e.content.MissingItemRange
}
