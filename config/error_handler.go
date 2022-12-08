package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Body   = &ErrorHandler{}
	_ Inline = &ErrorHandler{}
)

// ErrorHandler represents a subset of Endpoint.
type ErrorHandler struct {
	Kinds     []string
	ErrorFile string    `hcl:"error_file,optional" docs:"Location of the error file template."`
	Proxies   Proxies   `hcl:"proxy,block" docs:"Configures a [proxy](/configuration/block/proxy) (zero or more)."`
	Remain    hcl.Body  `hcl:",remain"`
	Requests  Requests  `hcl:"request,block" docs:"Configures a [request](/configuration/block/request) (zero or more)."`
	Response  *Response `hcl:"response,block" docs:"Configures the [response](/configuration/block/response) (zero or one)."`
}

// ErrorHandlerGetter defines the <ErrorHandlerGetter> interface.
type ErrorHandlerGetter interface {
	DefaultErrorHandler() *ErrorHandler
}

// HCLBody implements the <Body> interface.
func (e ErrorHandler) HCLBody() *hclsyntax.Body {
	return e.Remain.(*hclsyntax.Body)
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
