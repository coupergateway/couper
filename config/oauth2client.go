package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// OAuth2Client defines the <OAuth2Client> interface.
type OAuth2Client interface {
	Inline
	GetClientID() string
	GetClientSecret() string
	GetGrantType() string
	GetScope() *string
	GetTokenEndpointAuthMethod() *string
}

type OAuth2AcClient interface {
	OAuth2Client
	GetCsrf() *CSRF
	GetPkce() *PKCE
}

type PKCE struct {
	CodeChallengeMethod string   `hcl:"code_challenge_method"`
	Remain              hcl.Body `hcl:",remain"`
	// internally used
	Content *hcl.BodyContent
}

// HCLBody implements the <Body> interface.
func (p PKCE) HCLBody() hcl.Body {
	return p.Remain
}

// Schema implements the <Inline> interface.
func (p PKCE) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(p)
		return schema
	}

	type Inline struct {
		CodeVerifierValue string `hcl:"code_verifier_value"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}

type CSRF struct {
	TokenParam string   `hcl:"token_param"`
	Remain     hcl.Body `hcl:",remain"`
	// internally used
	Content *hcl.BodyContent
}

// HCLBody implements the <Body> interface.
func (c CSRF) HCLBody() hcl.Body {
	return c.Remain
}

// Schema implements the <Inline> interface.
func (c CSRF) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(c)
		return schema
	}

	type Inline struct {
		TokenValue string `hcl:"token_value"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	return schema
}
