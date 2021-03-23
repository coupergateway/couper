package config

import (
	"github.com/avenga/couper/config/meta"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// ErrorHandler represents the <ErrorHandler> object.
type ErrorHandler struct {
	Kinds  []string
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Inline> interface.
func (e ErrorHandler) HCLBody() hcl.Body {
	return e.Remain
}

// Schema implements the <Inline> interface.
func (e ErrorHandler) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		meta.ResponseAttributes
		Response *Response `hcl:"response,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
