package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var _ Inline = &Backend{}

// Backend represents the <Backend> object.
type Backend struct {
	BasicAuth              string   `hcl:"basic_auth,optional"`
	ConnectTimeout         string   `hcl:"connect_timeout,optional"`
	DisableCertValidation  bool     `hcl:"disable_certificate_validation,optional"`
	DisableConnectionReuse bool     `hcl:"disable_connection_reuse,optional"`
	HTTP2                  bool     `hcl:"http2,optional"`
	MaxConnections         int      `hcl:"max_connections,optional"`
	Name                   string   `hcl:"name,label"`
	OpenAPI                *OpenAPI `hcl:"openapi,block"`
	PathPrefix             string   `hcl:"path_prefix,optional"`
	Proxy                  string   `hcl:"proxy,optional"`
	Remain                 hcl.Body `hcl:",remain"`
	TTFBTimeout            string   `hcl:"ttfb_timeout,optional"`
	Timeout                string   `hcl:"timeout,optional"`
}

// HCLBody implements the <Inline> interface.
func (b Backend) HCLBody() hcl.Body {
	return b.Remain
}

// Reference implements the <Inline> interface.
func (b Backend) Reference() string {
	return b.Name
}

// Schema implements the <Inline> interface.
func (b Backend) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(b)
	if !inline {
		return schema
	}

	type Inline struct {
		Hostname           string               `hcl:"hostname,optional"`
		Origin             string               `hcl:"origin,optional"`
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

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})

	return schema
}

func newBackendSchema(schema *hcl.BodySchema, body hcl.Body) *hcl.BodySchema {
	for i, block := range schema.Blocks {
		// Inline backend block MAY have no label.
		if block.Type == "backend" && len(block.LabelNames) > 0 {
			// Check if a backend block could be parsed w/ label, otherwise its an inline one w/o label.
			content, _, _ := body.PartialContent(schema)
			if content == nil || len(content.Blocks) == 0 {
				schema.Blocks[i].LabelNames = nil

				break
			}
		}
	}

	return schema
}
