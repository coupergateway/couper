package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &ErrorHandler{}

// ErrorHandler represents the <ErrorHandler> object.
type ErrorHandler struct {
	Name     string    `hcl:"name,label"`
	Remain   hcl.Body  `hcl:",remain"`
	Response *Response `hcl:"response,block"`
}

// HCLBody implements the <Inline> interface.
func (e ErrorHandler) HCLBody() hcl.Body {
	return e.Remain
}

// Reference implements the <Inline> interface.
func (e ErrorHandler) Reference() string {
	return e.Name
}

// Schema implements the <Inline> interface.
func (e ErrorHandler) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		AddResponseHeaders map[string]string `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string          `hcl:"remove_response_headers,optional"`
		SetResponseHeaders map[string]string `hcl:"set_response_headers,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
