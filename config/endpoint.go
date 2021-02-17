package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Endpoint{}

// Endpoint represents the <Endpoint> object.
type Endpoint struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Pattern              string    `hcl:"pattern,label"`
	Proxies              Proxies   `hcl:"proxy,block"`
	Remain               hcl.Body  `hcl:",remain"`
	Requests             Requests  `hcl:"request,block"`
	Response             *Response `hcl:"response,block"`
}

// Endpoints represents a list of <Endpoint> objects.
type Endpoints []*Endpoint

// HCLBody implements the <Inline> interface.
func (e Endpoint) HCLBody() hcl.Body {
	return e.Remain
}

// Reference implements the <Inline> interface.
func (e Endpoint) Reference() string {
	return e.Pattern
}

// Schema implements the <Inline> interface.
func (e Endpoint) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	type Inline struct {
		Path               string               `hcl:"path,optional"`
		SetRequestHeaders  map[string]string    `hcl:"set_request_headers,optional"`
		AddRequestHeaders  map[string]string    `hcl:"add_request_headers,optional"`
		DelRequestHeaders  []string             `hcl:"remove_request_headers,optional"`
		SetResponseHeaders map[string]string    `hcl:"set_response_headers,optional"`
		AddResponseHeaders map[string]string    `hcl:"add_response_headers,optional"`
		DelResponseHeaders []string             `hcl:"remove_response_headers,optional"`
		AddQueryParams     map[string]cty.Value `hcl:"add_query_params,optional"`
		DelQueryParams     []string             `hcl:"remove_query_params,optional"`
		SetQueryParams     map[string]cty.Value `hcl:"set_query_params,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
