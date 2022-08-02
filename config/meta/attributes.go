package meta

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var AttributesSchema, _ = gohcl.ImpliedBodySchema(&Attributes{})

// Attributes are commonly shared attributes which gets evaluated during runtime.
type Attributes struct {
	// Form Params
	AddFormParams map[string]cty.Value `hcl:"add_form_params,optional" docs:"key/value pairs to add form parameters to the upstream request body"`
	DelFormParams map[string]cty.Value `hcl:"remove_form_params,optional" docs:"list of names to remove form parameters from the upstream request body"`
	SetFormParams map[string]cty.Value `hcl:"set_form_params,optional" docs:"key/value pairs to set query parameters in the upstream request URL"`

	// Query Params
	AddQueryParams map[string]cty.Value `hcl:"add_query_params,optional" docs:"key/value pairs to add query parameters to the upstream request URL"`
	DelQueryParams []string             `hcl:"remove_query_params,optional" docs:"list of names to remove query parameters from the upstream request URL"`
	SetQueryParams map[string]cty.Value `hcl:"set_query_params,optional" docs:"key/value pairs to set query parameters in the upstream request URL"`

	// Request Header Modifiers
	AddRequestHeaders map[string]string `hcl:"add_request_headers,optional" docs:"key/value pairs to add as request headers in the upstream request"`
	DelRequestHeaders []string          `hcl:"remove_request_headers,optional" docs:"list of names to remove headers from the upstream request"`
	SetRequestHeaders map[string]string `hcl:"set_request_headers,optional" docs:"key/value pairs to set as request headers in the upstream request"`

	// Response Header Modifiers
	AddResponseHeaders map[string]string `hcl:"add_response_headers,optional" docs:"key/value pairs to add as response headers in the client response"`
	DelResponseHeaders []string          `hcl:"remove_response_headers,optional" docs:"list of names to remove headers from the client response"`
	SetResponseHeaders map[string]string `hcl:"set_response_headers,optional" docs:"key/value pairs to set as response headers in the client response"`
}

func SchemaWithAttributes(schema *hcl.BodySchema) *hcl.BodySchema {
	schema.Attributes = append(schema.Attributes, AttributesSchema.Attributes...)
	schema.Blocks = append(schema.Blocks, AttributesSchema.Blocks...)

	return schema
}
