package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ OAuth2AcClient = &OIDC{}
var _ OAuth2AS = &OIDC{}
var _ OidcAS = &OIDC{}

// OIDC represents an oidc block.
type OIDC struct {
	AccessControlSetter
	AuthorizationEndpoint   string   `hcl:"authorization_endpoint"`
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	Csrf                    *CSRF    `hcl:"csrf,block"`
	Issuer                  string   `hcl:"issuer"`
	Name                    string   `hcl:"name,label"`
	Pkce                    *PKCE    `hcl:"pkce,block"`
	RedirectURI             *string  `hcl:"redirect_uri"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpoint           string   `hcl:"token_endpoint"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	UserinfoEndpoint        string   `hcl:"userinfo_endpoint"`
	// internally used
	Backend hcl.Body
}

func (o OIDC) HCLBody() hcl.Body {
	return o.Remain
}

func (o OIDC) Reference() string {
	return o.BackendName
}

func (o OIDC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(o)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if o.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, o.HCLBody())
}

func (o OIDC) GetName() string {
	return o.Name
}

func (o OIDC) GetClientID() string {
	return o.ClientID
}

func (o OIDC) GetClientSecret() string {
	return o.ClientSecret
}

func (o OIDC) GetGrantType() string {
	return "authorization_code"
}

func (o OIDC) GetScope() *string {
	return o.Scope
}

func (o OIDC) GetRedirectURI() *string {
	return o.RedirectURI
}

func (o OIDC) GetTokenEndpoint() string {
	return o.TokenEndpoint
}

func (o OIDC) GetTokenEndpointAuthMethod() *string {
	return o.TokenEndpointAuthMethod
}

func (o OIDC) GetCsrf() *CSRF {
	return o.Csrf
}

func (o OIDC) GetPkce() *PKCE {
	return o.Pkce
}

func (o OIDC) GetIssuer() string {
	return o.Issuer
}

func (o OIDC) GetUserinfoEndpoint() string {
	return o.UserinfoEndpoint
}
