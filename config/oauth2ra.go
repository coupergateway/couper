package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var OAuthBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "oauth2",
		},
	},
}

var _ OAuth2Client = &OAuth2ReqAuth{}
var _ OAuth2AS = &OAuth2ReqAuth{}

// OAuth2ReqAuth represents the <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	GrantType               string   `hcl:"grant_type"`
	Remain                  hcl.Body `hcl:",remain"`
	Retries                 *uint8   `hcl:"retries,optional"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpoint           string   `hcl:"token_endpoint,optional"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
}

func (oa OAuth2ReqAuth) HCLBody() hcl.Body {
	return oa.Remain
}

func (oa OAuth2ReqAuth) Reference() string {
	return oa.BackendName
}

func (oa OAuth2ReqAuth) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}

func (oa OAuth2ReqAuth) GetClientID() string {
	return oa.ClientID
}

func (oa OAuth2ReqAuth) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa OAuth2ReqAuth) GetGrantType() string {
	return oa.GrantType
}

func (oa OAuth2ReqAuth) GetScope() *string {
	return oa.Scope
}

func (oa OAuth2ReqAuth) GetTokenEndpoint() string {
	return oa.TokenEndpoint
}

func (oa OAuth2ReqAuth) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}
