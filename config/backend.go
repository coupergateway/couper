package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference = &Backend{}
	_ Body             = &Backend{}
	_ Inline           = &Backend{}

	BackendInlineSchema = Backend{}.Schema(true)
)

// Backend represents the <Backend> object.
type Backend struct {
	DisableCertValidation  bool       `hcl:"disable_certificate_validation,optional" docs:"Disables the peer certificate validation. Must not be used in backend refinement."`
	DisableConnectionReuse bool       `hcl:"disable_connection_reuse,optional" docs:"Disables reusage of connections to the origin. Must not be used in backend refinement."`
	Health                 *Health    `hcl:"beta_health,block"`
	HTTP2                  bool       `hcl:"http2,optional" docs:"Enables the HTTP2 support. Must not be used in backend refinement."`
	MaxConnections         int        `hcl:"max_connections,optional" docs:"The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. Must not be used in backend refinement." default:"0"`
	Name                   string     `hcl:"name,label,optional"`
	OpenAPI                *OpenAPI   `hcl:"openapi,block"`
	RateLimits             RateLimits `hcl:"beta_rate_limit,block"`
	Remain                 hcl.Body   `hcl:",remain"`
}

// Reference implements the <BackendReference> interface.
func (b Backend) Reference() string {
	return b.Name
}

// HCLBody implements the <Body> interface.
func (b Backend) HCLBody() *hclsyntax.Body {
	return b.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (b Backend) Inline() interface{} {
	type Inline struct {
		meta.RequestHeadersAttributes
		meta.ResponseHeadersAttributes
		meta.FormParamsAttributes
		meta.QueryParamsAttributes
		meta.LogFieldsAttribute
		BasicAuth      string `hcl:"basic_auth,optional" docs:"Basic auth for the upstream request with format user:pass ."`
		ConnectTimeout string `hcl:"connect_timeout,optional" docs:"The total timeout for dialing and connect to the origin." type:"duration" default:"10s"`
		Hostname       string `hcl:"hostname,optional" docs:"Value of the HTTP host header field for the origin request. Since hostname replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin."`
		Origin         string `hcl:"origin,optional" docs:"URL to connect to for backend requests."`
		Path           string `hcl:"path,optional" docs:"Changeable part of upstream URL."`
		PathPrefix     string `hcl:"path_prefix,optional" docs:"Prefixes all backend request paths with the given prefix"`
		ProxyURL       string `hcl:"proxy,optional" docs:"A proxy URL for the related origin request."`
		ResponseStatus *uint8 `hcl:"set_response_status,optional" docs:"Modifies the response status code."`
		TTFBTimeout    string `hcl:"ttfb_timeout,optional" docs:"The duration from writing the full request to the origin and receiving the answer." type:"duration" default:"60s"`
		Timeout        string `hcl:"timeout,optional" docs:"The total deadline duration a backend request has for write and read/pipe." type:"duration" default:"300s"`
		UseUnhealthy   bool   `hcl:"use_when_unhealthy,optional" docs:"Ignores the health state and continues with the outgoing request"`

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

	return meta.MergeSchemas(schema, meta.ModifierAttributesSchema, meta.LogFieldsAttributeSchema)
}
