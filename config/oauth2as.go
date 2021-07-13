package config

// OAuth2AS represents the OAuth2 authorization server configuration.
type OAuth2AS interface {
	BackendReference
	GetTokenEndpoint() string
}

// OidcAS represents the OIDC server configuration.
type OidcAS interface {
	OAuth2AS
	GetIssuer() string
	GetUserinfoEndpoint() string
}
