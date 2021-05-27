package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ OAuth2Config = &OAuth2ReqAuth{}
)

// OAuth2ReqAuth represents the <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	BackendName string   `hcl:"backend,optional"`
	GrantType   string   `hcl:"grant_type"`
	Remain      hcl.Body `hcl:",remain"`
	Retries     *uint8   `hcl:"retries,optional"`
}

// HCLBody implements the <Body> interface.
func (oa OAuth2ReqAuth) HCLBody() hcl.Body {
	return oa.Remain
}

// Reference implements the <BackendReference> interface.
func (oa OAuth2ReqAuth) Reference() string {
	return oa.BackendName
}

// Schema implements the <Inline> interface.
func (oa OAuth2ReqAuth) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend                 *Backend `hcl:"backend,block"`
		ClientID                string   `hcl:"client_id"`
		ClientSecret            string   `hcl:"client_secret"`
		Scope                   *string  `hcl:"scope,optional"`
		TokenEndpoint           string   `hcl:"token_endpoint,optional"`
		TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}

// GetGrantType implements the <OAuth2Config> interface.
func (oa OAuth2ReqAuth) GetGrantType() string {
	return oa.GrantType
}
