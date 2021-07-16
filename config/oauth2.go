package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// OAuth2AS represents the authorization server configuration for OAuth2 clients.
type OAuth2AS interface {
	BackendReference
	GetTokenEndpoint() (string, error)
}

// OAuth2AcAS represents the authorization server configuration for OAuth2 clients using the authorization code flow.
type OAuth2AcAS interface {
	OAuth2AS
	GetAuthorizationEndpoint() (string, error)
}

// OidcAS represents the OIDC server configuration for OIDC clients.
type OidcAS interface {
	OAuth2AcAS
	GetIssuer() (string, error)
	GetUserinfoEndpoint() (string, error)
}

// OAuth2Client represents the client configuration for OAuth2 clients.
type OAuth2Client interface {
	Inline
	GetClientID() string
	GetClientSecret() string
	GetGrantType() string
	GetScope() string
	GetTokenEndpointAuthMethod() *string
}

// OAuth2AcClient represents the client configuration for OAuth2 clients using the authorization code flow.
type OAuth2AcClient interface {
	OAuth2Client
	GetName() string
	GetCsrf() *CSRF
	GetPkce() *PKCE
	GetRedirectURI() *string
}

// OAuth2Authorization represents the configuration for the OAuth2 authorization URL function
type OAuth2Authorization interface {
	GetAuthorizationEndpoint() (string, error)
	GetClientID() string
	GetCsrf() *CSRF
	GetName() string
	GetPkce() *PKCE
	GetRedirectURI() *string
	GetScope() string
}

// PKCE represents the PKCE configuration.
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

// CSRF represents the CSRF configuration.
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
