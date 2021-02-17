package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Proxy{}

// Proxy represents the <Proxy> object.
type Proxy struct {
	Backend string   `hcl:"backend,optional"`
	Remain  hcl.Body `hcl:",remain"`
}

// Proxies represents a list of <Proxy> objects.
type Proxies []*Proxy

// HCLBody implements the <Inline> interface.
func (p Proxy) HCLBody() hcl.Body {
	return p.Remain
}

// Reference implements the <Inline> interface.
func (p Proxy) Reference() string {
	return "proxy"
}

// Schema implements the <Inline> interface.
func (p Proxy) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	type Inline struct {
		Backend            *Backend             `hcl:"backend,block"`
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

	// A backend reference is defined, backend block is not allowed.
	if p.Backend != "" {
		schema.Blocks = nil
	}

	// TODO: Wenn <URL> definiert, dann kein <Backend> und <Path>

	return newBackendSchema(schema, p.HCLBody())
}
