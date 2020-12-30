package body

import "github.com/hashicorp/hcl/v2"

var _ hcl.Body = &Body{}

type Body struct {
	hcl.BodyContent
}

func New(content *hcl.BodyContent) hcl.Body {
	return &Body{*content}
}

func NewFromAttributes(body hcl.Body) hcl.Body {
	attrs, _ := body.JustAttributes()
	content := &hcl.BodyContent{
		Attributes:       attrs,
		MissingItemRange: body.MissingItemRange(),
	}
	return New(content)
}

func (e *Body) Content(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	return &e.BodyContent, nil
}

func (e *Body) PartialContent(_ *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	return &e.BodyContent, e, nil
}

func (e *Body) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	return e.Attributes, nil
}

func (e *Body) MissingItemRange() hcl.Range {
	return e.BodyContent.MissingItemRange
}
