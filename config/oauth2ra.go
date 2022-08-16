package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

const (
	ClientCredentials = "client_credentials"
	JwtBearer         = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	Password          = "password"
)

var OAuthBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "oauth2",
		},
	},
}

var (
	_ BackendReference = &OAuth2ReqAuth{}
	_ Inline           = &OAuth2ReqAuth{}
	_ OAuth2Client     = &OAuth2ReqAuth{}
	_ OAuth2AS         = &OAuth2ReqAuth{}
)

// OAuth2ReqAuth represents the oauth2 block in a backend block.
type OAuth2ReqAuth struct {
	AssertionExpr           hcl.Expression `hcl:"assertion,optional"`
	BackendName             string         `hcl:"backend,optional"`
	ClientID                string         `hcl:"client_id,optional"`
	ClientSecret            string         `hcl:"client_secret,optional"`
	GrantType               string         `hcl:"grant_type"`
	Password                string         `hcl:"password,optional"`
	Remain                  hcl.Body       `hcl:",remain"`
	Retries                 *uint8         `hcl:"retries,optional"`
	Scope                   string         `hcl:"scope,optional"`
	TokenEndpoint           string         `hcl:"token_endpoint,optional"`
	TokenEndpointAuthMethod *string        `hcl:"token_endpoint_auth_method,optional"`
	Username                string         `hcl:"username,optional"`
}

// Reference implements the <BackendReference> interface.
func (oa *OAuth2ReqAuth) Reference() string {
	return oa.BackendName
}

// HCLBody implements the <Inline> interface.
func (oa *OAuth2ReqAuth) HCLBody() hcl.Body {
	return oa.Remain
}

// Inline implements the <Inline> interface.
func (oa *OAuth2ReqAuth) Inline() interface{} {
	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (oa *OAuth2ReqAuth) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(oa.Inline())

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return schema
}

func (oa *OAuth2ReqAuth) ClientAuthenticationRequired() bool {
	return oa.GrantType != JwtBearer
}

func (oa *OAuth2ReqAuth) GetClientID() string {
	return oa.ClientID
}

func (oa *OAuth2ReqAuth) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa *OAuth2ReqAuth) GetTokenEndpoint() (string, error) {
	return oa.TokenEndpoint, nil
}

func (oa *OAuth2ReqAuth) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}
