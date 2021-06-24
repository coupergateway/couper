package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ OAuth2 = &OAuth2AC{}

// OAuth2AC represents the <OAuth2> access control object.
type OAuth2AC struct {
	AccessControlSetter
	AuthorizationEndpoint   string   `hcl:"authorization_endpoint"`
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	Csrf                    *CSRF    `hcl:"csrf,block"`
	GrantType               string   `hcl:"grant_type"`
	Issuer                  string   `hcl:"issuer,optional"`
	Name                    string   `hcl:"name,label"`
	Pkce                    *PKCE    `hcl:"pkce,block"`
	RedirectURI             *string  `hcl:"redirect_uri"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	UserinfoEndpoint        string   `hcl:"userinfo_endpoint,optional"`
	// internally used
	Backend hcl.Body
}

func (oa OAuth2AC) HCLBody() hcl.Body {
	return oa.Remain
}

func (oa OAuth2AC) Reference() string {
	return oa.BackendName
}

func (oa OAuth2AC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend       *Backend `hcl:"backend,block"`
		TokenEndpoint string   `hcl:"token_endpoint,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}

func (oa OAuth2AC) GetClientID() string {
	return oa.ClientID
}

func (oa OAuth2AC) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa OAuth2AC) GetGrantType() string {
	return oa.GrantType
}

func (oa OAuth2AC) GetScope() *string {
	return oa.Scope
}

func (oa OAuth2AC) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}

type PKCE struct {
	CodeChallengeMethod string   `hcl:"code_challenge_method"`
	Remain              hcl.Body `hcl:",remain"`
	// internally used
	Content *hcl.BodyContent
}

// HCLBody implements the <Body> interface.
func (p PKCE) HCLBody() hcl.Body {
	return p.Remain
}

// Schema implements the <Inline> interface.
func (p PKCE) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	type Inline struct {
		CodeVerifierValue string `hcl:"code_verifier_value"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}

type CSRF struct {
	TokenParam string   `hcl:"token_param"`
	Remain     hcl.Body `hcl:",remain"`
	// internally used
	Content *hcl.BodyContent
}

// HCLBody implements the <Body> interface.
func (c CSRF) HCLBody() hcl.Body {
	return c.Remain
}

// Schema implements the <Inline> interface.
func (c CSRF) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(c)
		return schema
	}

	type Inline struct {
		TokenValue string `hcl:"token_value"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
