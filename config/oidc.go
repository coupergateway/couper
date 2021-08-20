package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// OIDC represents an oidc block.
type OIDC struct {
	AccessControlSetter
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	ConfigurationURL        string   `hcl:"configuration_url"`
	Name                    string   `hcl:"name,label"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	ConfigurationTTL        string   `hcl:"configuration_ttl,optional"`
	VerifierMethod          string   `hcl:"verifier_method,optional"`

	// internally used
	Backend     hcl.Body
	BodyContent *hcl.BodyContent
}

func (o OIDC) HCLBody() hcl.Body {
	return o.Remain
}

func (o OIDC) Reference() string {
	return o.BackendName
}

func (o *OIDC) GetBodyContent() *hcl.BodyContent {
	return o.BodyContent
}

func (o OIDC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(o)
		return schema
	}

	type Inline struct {
		Backend       *Backend `hcl:"backend,block"`
		RedirectURI   string   `hcl:"redirect_uri"`
		VerifierValue string   `hcl:"verifier_value"`
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

func (o OIDC) GetScope() string {
	if o.Scope == nil {
		return "openid"
	}
	return "openid " + *o.Scope
}

func (o OIDC) GetTokenEndpointAuthMethod() *string {
	return o.TokenEndpointAuthMethod
}
