package meta

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var RequestHeadersAttributesSchema, _ = gohcl.ImpliedBodySchema(&RequestHeadersAttributes{})
var ResponseHeadersAttributesSchema, _ = gohcl.ImpliedBodySchema(&ResponseHeadersAttributes{})
var FormParamsAttributesSchema, _ = gohcl.ImpliedBodySchema(&FormParamsAttributes{})
var QueryParamsAttributesSchema, _ = gohcl.ImpliedBodySchema(&QueryParamsAttributes{})
var LogFieldsAttributeSchema, _ = gohcl.ImpliedBodySchema(&LogFieldsAttribute{})

var ModifierAttributesSchema = MergeSchemas(
	RequestHeadersAttributesSchema,
	ResponseHeadersAttributesSchema,
	FormParamsAttributesSchema,
	QueryParamsAttributesSchema,
)

// Attributes are commonly shared attributes which gets evaluated during runtime.

type FormParamsAttributes struct {
	// Form Params
	AddFormParams map[string]cty.Value `hcl:"add_form_params,optional" docs:"Key/value pairs to add form parameters to the upstream request body."`
	DelFormParams map[string]cty.Value `hcl:"remove_form_params,optional" docs:"List of names to remove form parameters from the upstream request body."`
	SetFormParams map[string]cty.Value `hcl:"set_form_params,optional" docs:"Key/value pairs to set query parameters in the upstream request URL."`
}

type QueryParamsAttributes struct {
	// Query Params
	AddQueryParams map[string]cty.Value `hcl:"add_query_params,optional" docs:"Key/value pairs to add query parameters to the upstream request URL."`
	DelQueryParams []string             `hcl:"remove_query_params,optional" docs:"List of names to remove query parameters from the upstream request URL."`
	SetQueryParams map[string]cty.Value `hcl:"set_query_params,optional" docs:"Key/value pairs to set query parameters in the upstream request URL."`
}

type RequestHeadersAttributes struct {
	// Request Header Modifiers
	AddRequestHeaders map[string]string `hcl:"add_request_headers,optional" docs:"Key/value pairs to add as request headers in the upstream request."`
	DelRequestHeaders []string          `hcl:"remove_request_headers,optional" docs:"List of names to remove headers from the upstream request."`
	SetRequestHeaders map[string]string `hcl:"set_request_headers,optional" docs:"Key/value pairs to set as request headers in the upstream request."`
}

type ResponseHeadersAttributes struct {
	// Response Header Modifiers
	AddResponseHeaders map[string]string `hcl:"add_response_headers,optional" docs:"Key/value pairs to add as response headers in the client response."`
	DelResponseHeaders []string          `hcl:"remove_response_headers,optional" docs:"List of names to remove headers from the client response."`
	SetResponseHeaders map[string]string `hcl:"set_response_headers,optional" docs:"Key/value pairs to set as response headers in the client response."`
}

type LogFieldsAttribute struct {
	LogFields map[string]hcl.Expression `hcl:"custom_log_fields,optional" docs:"Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."`
}

func MergeSchemas(schema *hcl.BodySchema, schemas ...*hcl.BodySchema) *hcl.BodySchema {
	for _, s := range schemas {
		schema.Attributes = append(schema.Attributes, s.Attributes...)
		schema.Blocks = append(schema.Blocks, s.Blocks...)
	}
	return schema
}
