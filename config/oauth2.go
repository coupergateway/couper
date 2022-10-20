package config

const (
	CcmS256 = "ccm_s256"
)

// OAuth2AS represents the authorization server configuration for OAuth2 clients.
type OAuth2AS interface {
	GetTokenEndpoint() (string, error)
}

// OAuth2Client represents the client configuration for OAuth2 clients.
type OAuth2Client interface {
	ClientAuthenticationRequired() bool
	GetClientID() string
	GetClientSecret() string
	GetTokenEndpointAuthMethod() *string
}

// OAuth2AcClient represents the client configuration for OAuth2 clients using the authorization code flow.
type OAuth2AcClient interface {
	Body
	OAuth2Client
	GetGrantType() string
	GetRedirectURI() string
	// GetVerifierMethod retrieves the verifier method (ccm_s256, nonce or state)
	GetVerifierMethod() (string, error)
}

// OAuth2Authorization represents the configuration for the OAuth2 authorization URL function
type OAuth2Authorization interface {
	GetAuthorizationEndpoint() (string, error)
	GetClientID() string
	GetRedirectURI() string
	GetScope() string
	GetVerifierMethod() (string, error)
}
