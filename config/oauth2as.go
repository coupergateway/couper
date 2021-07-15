package config

// OAuth2AS represents the OAuth2 authorization server configuration.
type OAuth2AS interface {
	BackendReference
	GetTokenEndpoint() (string, error)
}

// OAuth2AcAS represents the authorization server configuration for authorization code clients.
type OAuth2AcAS interface {
	OAuth2AS
	GetAuthorizationEndpoint() (string, error)
}

// OidcAS represents the OIDC server configuration.
type OidcAS interface {
	OAuth2AcAS
	GetIssuer() (string, error)
	GetUserinfoEndpoint() (string, error)
}
