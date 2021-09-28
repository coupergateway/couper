package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/health_check"
	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference = &Backend{}
	_ Inline           = &Backend{}

	BackendInlineSchema = Backend{}.Schema(true)
)

// Backend represents the <Backend> object.
type Backend struct {
	DisableCertValidation  bool                      `hcl:"disable_certificate_validation,optional"`
	DisableConnectionReuse bool                      `hcl:"disable_connection_reuse,optional"`
	HealthCheck            *health_check.HealthCheck `hcl:"health_check,block"`
	HTTP2                  bool                      `hcl:"http2,optional"`
	MaxConnections         int                       `hcl:"max_connections,optional"`
	Name                   string                    `hcl:"name,label"`
	OpenAPI                *OpenAPI                  `hcl:"openapi,block"`
	Remain                 hcl.Body                  `hcl:",remain"`

	// explicit configuration on load
	OAuth2 *OAuth2ReqAuth
}

// Reference implements the <BackendReference> interface.
func (b Backend) Reference() string {
	return b.Name
}

// HCLBody implements the <Inline> interface.
func (b Backend) HCLBody() hcl.Body {
	return b.Remain
}

// Inline implements the <Inline> interface.
func (b Backend) Inline() interface{} {
	type Inline struct {
		meta.Attributes
		BasicAuth      string                    `hcl:"basic_auth,optional"`
		ConnectTimeout string                    `hcl:"connect_timeout,optional"`
		Hostname       string                    `hcl:"hostname,optional"`
		LogFields      map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
		Origin         string                    `hcl:"origin,optional"`
		PathPrefix     string                    `hcl:"path_prefix,optional"`
		ProxyURL       string                    `hcl:"proxy,optional"`
		ResponseStatus *uint8                    `hcl:"set_response_status,optional"`
		TTFBTimeout    string                    `hcl:"ttfb_timeout,optional"`
		Timeout        string                    `hcl:"timeout,optional"`

		// set by backend preparation
		BackendURL string `hcl:"backend_url,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (b Backend) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(b)
	if !inline {
		return schema
	}

	schema, _ = gohcl.ImpliedBodySchema(b.Inline())

	return meta.SchemaWithAttributes(schema)
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
			if content == nil || len(content.Blocks.OfType("backend")) == 0 {
				schema.Blocks[i].LabelNames = nil
				break
			}
		}
	}

	return schema
}
