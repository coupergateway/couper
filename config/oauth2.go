package config

import (
	"github.com/hashicorp/hcl/v2"
)

const (
	CcmS256 = "ccm_s256"
)

// OAuth2AS represents the authorization server configuration for OAuth2 clients.
type OAuth2AS interface {
	BackendReference
	GetTokenEndpoint() (string, error)
}

// OAuth2AcAS represents the authorization server configuration for OAuth2 clients using the authorization code flow.
type OAuth2AcAS interface {
	OAuth2AS
	GetAuthorizationEndpoint(uid string) (string, error)
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
	// GetVerifierMethod retrieves the verifier method (ccm_s256, nonce or state)
	GetVerifierMethod(uid string) (string, error)
	GetBodyContent() *hcl.BodyContent
}

// OAuth2Authorization represents the configuration for the OAuth2 authorization URL function
type OAuth2Authorization interface {
	Inline
	GetAuthorizationEndpoint(uid string) (string, error)
	GetClientID() string
	GetName() string
	GetScope() string
	GetVerifierMethod(uid string) (string, error)
}
