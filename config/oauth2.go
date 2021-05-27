package config

import (
	"github.com/hashicorp/hcl/v2"
)

// OAuth2Config defines the <OAuth2Config> interface.
type OAuth2Config interface {
	BackendReference
	Inline
	GetGrantType() string
}

var OAuthBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "oauth2",
		},
	},
}
