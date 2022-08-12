package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &ErrorHandler{}

// ErrorHandler represents a subset of Endpoint.
type ErrorHandler struct {
	Kinds     []string
	ErrorFile string    `hcl:"error_file,optional"`
	Proxies   Proxies   `hcl:"proxy,block"`
	Remain    hcl.Body  `hcl:",remain"`
	Requests  Requests  `hcl:"request,block"`
	Response  *Response `hcl:"response,block"`
}

// ErrorHandlerGetter defines the <ErrorHandlerGetter> interface.
type ErrorHandlerGetter interface {
	DefaultErrorHandler() *ErrorHandler
}

// HCLBody implements the <Inline> interface.
func (e ErrorHandler) HCLBody() hcl.Body {
	return e.Remain
}

// Inline implements the <Inline> interface.
func (e ErrorHandler) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		meta.FormParamsAttributes
		meta.QueryParamsAttributes
		meta.LogFieldsAttribute
		ResponseStatus *uint8 `hcl:"set_response_status,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (e ErrorHandler) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(e.Inline())
	return meta.MergeSchemas(schema, meta.ModifierAttributesSchema, meta.LogFieldsAttributeSchema)
}
