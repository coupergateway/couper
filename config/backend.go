package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ Inline           = &Backend{}
	_ BackendReference = &Backend{}

	BackendInlineSchema = Backend{}.Schema(true)
)

// Backend represents the <Backend> object.
type Backend struct {
	ConnectTimeout         string   `hcl:"connect_timeout,optional"`
	DisableCertValidation  bool     `hcl:"disable_certificate_validation,optional"`
	DisableConnectionReuse bool     `hcl:"disable_connection_reuse,optional"`
	HTTP2                  bool     `hcl:"http2,optional"`
	MaxConnections         int      `hcl:"max_connections,optional"`
	Name                   string   `hcl:"name,label"`
	OpenAPI                *OpenAPI `hcl:"openapi,block"`
	Remain                 hcl.Body `hcl:",remain"`
	TTFBTimeout            string   `hcl:"ttfb_timeout,optional"`
	Timeout                string   `hcl:"timeout,optional"`

	// explicit configuration on load
	OAuth2 *OAuth2ReqAuth
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
		meta.Attributes
		BasicAuth      string `hcl:"basic_auth,optional"`
		Hostname       string `hcl:"hostname,optional"`
		Origin         string `hcl:"origin,optional"`
		PathPrefix     string `hcl:"path_prefix,optional"`
		ProxyURL       string `hcl:"proxy,optional"`
		ResponseStatus *uint8 `hcl:"set_response_status,optional"`
	}

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})

	return schema
}

func newBackendSchema(schema *hcl.BodySchema, body hcl.Body) *hcl.BodySchema {
	if body == nil {
		return schema
	}

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
