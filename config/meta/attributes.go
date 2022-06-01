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
	AddFormParams map[string]cty.Value `hcl:"add_form_params,optional"`
	DelFormParams map[string]cty.Value `hcl:"remove_form_params,optional"`
	SetFormParams map[string]cty.Value `hcl:"set_form_params,optional"`

	// Query Params
	AddQueryParams map[string]cty.Value `hcl:"add_query_params,optional"`
	DelQueryParams []string             `hcl:"remove_query_params,optional"`
	SetQueryParams map[string]cty.Value `hcl:"set_query_params,optional"`

	// Request Header Modifiers
	AddRequestHeaders map[string]string `hcl:"add_request_headers,optional"`
	DelRequestHeaders []string          `hcl:"remove_request_headers,optional"`
	SetRequestHeaders map[string]string `hcl:"set_request_headers,optional"`

	// Response Header Modifiers
	AddResponseHeaders map[string]string `hcl:"add_response_headers,optional"`
	DelResponseHeaders []string          `hcl:"remove_response_headers,optional"`
	SetResponseHeaders map[string]string `hcl:"set_response_headers,optional"`
}

func SchemaWithAttributes(schema *hcl.BodySchema) *hcl.BodySchema {
	schema.Attributes = append(schema.Attributes, AttributesSchema.Attributes...)
	schema.Blocks = append(schema.Blocks, AttributesSchema.Blocks...)

	return schema
}
