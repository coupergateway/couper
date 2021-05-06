package meta

import (
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var AttributesSchema, _ = gohcl.ImpliedBodySchema(&Attributes{})

// Attributes are commonly shared attributes which gets evaluated during runtime.
type Attributes struct {
	// RequestAttributes
	AddQueryParams    map[string]cty.Value `hcl:"add_query_params,optional"`
	AddRequestHeaders map[string]string    `hcl:"add_request_headers,optional"`
	DelQueryParams    []string             `hcl:"remove_query_params,optional"`
	DelRequestHeaders []string             `hcl:"remove_request_headers,optional"`
	Path              string               `hcl:"path,optional"`
	SetQueryParams    map[string]cty.Value `hcl:"set_query_params,optional"`
	SetRequestHeaders map[string]string    `hcl:"set_request_headers,optional"`
	// ResponseAttributes
	AddResponseHeaders map[string]string `hcl:"add_response_headers,optional"`
	DelResponseHeaders []string          `hcl:"remove_response_headers,optional"`
	SetResponseHeaders map[string]string `hcl:"set_response_headers,optional"`
}
