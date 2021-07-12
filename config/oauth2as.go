package config

// OAuth2AS defines the <OAuth2AS> interface.
type OAuth2AS interface {
	BackendReference
	GetTokenEndpoint() string
}
