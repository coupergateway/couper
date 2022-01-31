package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ BackendReference = &OIDC{}
	_ Inline           = &OIDC{}
)

// OIDC represents an oidc block.
type OIDC struct {
	ErrorHandlerSetter
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
	Backend hcl.Body
}

// Reference implements the <BackendReference> interface.
func (o *OIDC) Reference() string {
	return o.BackendName
}

// HCLBody implements the <Inline> interface.
func (o *OIDC) HCLBody() hcl.Body {
	return o.Remain
}

// Inline implements the <Inline> interface.
func (o *OIDC) Inline() interface{} {
	type Inline struct {
		Backend       *Backend                  `hcl:"backend,block"`
		LogFields     map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
		RedirectURI   string                    `hcl:"redirect_uri"`
		VerifierValue string                    `hcl:"verifier_value"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (o *OIDC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(o)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(o.Inline())

	// A backend reference is defined, backend block is not allowed.
	if o.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, o.HCLBody())
}

func (o *OIDC) GetName() string {
	return o.Name
}

func (o *OIDC) GetClientID() string {
	return o.ClientID
}

func (o *OIDC) GetClientSecret() string {
	return o.ClientSecret
}

func (o *OIDC) GetGrantType() string {
	return "authorization_code"
}

func (o *OIDC) GetScope() string {
	if o.Scope == nil {
		return "openid"
	}
	return "openid " + *o.Scope
}

func (o *OIDC) GetTokenEndpointAuthMethod() *string {
	return o.TokenEndpointAuthMethod
}
