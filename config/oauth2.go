package config

import (
	"github.com/hashicorp/hcl/v2"
)

// OAuth2 defines the <OAuth2> interface.
type OAuth2 interface {
	BackendReference
	Inline
	GetClientID() string
	GetClientSecret() string
	GetGrantType() string
	GetScope() *string
	GetTokenEndpoint() string
	GetTokenEndpointAuthMethod() *string
}

var OAuthBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "oauth2",
		},
	},
}
