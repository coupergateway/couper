package config

// OAuth2Client defines the <OAuth2Client> interface.
type OAuth2Client interface {
	Inline
	GetClientID() string
	GetClientSecret() string
	GetGrantType() string
	GetScope() *string
	GetTokenEndpointAuthMethod() *string
}
