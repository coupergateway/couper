package config

import (
	"github.com/hashicorp/hcl/v2"
)

// OAuth2 defines the <OAuth2> interface.
type OAuth2 interface {
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
