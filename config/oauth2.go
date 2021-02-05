package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ Inline = &OAuth2{}
)

// OAuth2 represents the "oauth2" config block.
type OAuth2 struct {
	// TODO
	// Name          string   `hcl:"name,label"`
	Remain        hcl.Body `hcl:",remain"`
	TokenEndpoint string   `hcl:"token_endpoint"`

	// TODO?
	// GrantType   string `hcl:"grant_type"`                 // Default: client_credentials
	// Scope       string `hcl:"scope"`                      // Default:
	// AuthMethod  string `hcl:"token_endpoint_auth_method"` // Default: client_secret_basic
}

func (oa OAuth2) Body() hcl.Body {
	return oa.Remain
}

func (oa OAuth2) Reference() string {
	return "OAuth2"
	// TODO return oa.Name
}

func (oa OAuth2) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(oa)
	if !inline {
		return schema
	}

	type Inline struct {
		Backend      *Backend `hcl:"backend,block"`
		ClientID     string   `hcl:"client_id"`
		ClientSecret string   `hcl:"client_secret"`
	}

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
