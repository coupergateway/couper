package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ OAuth2Config = &OAuth2AC{}
)

var _ Body = &OAuth2AC{}

// OAuth2AC represents the <OAuth2> access control object.
type OAuth2AC struct {
	AccessControlSetter
	BackendName string   `hcl:"backend,optional"`
	GrantType   string   `hcl:"grant_type"`
	Name        string   `hcl:"name,label"`
	Remain      hcl.Body `hcl:",remain"`
	// internally used
	Backend hcl.Body
}

// HCLBody implements the <Body> interface.
func (oa OAuth2AC) HCLBody() hcl.Body {
	return oa.Remain
}

// Reference implements the <Inline> interface.
func (oa OAuth2AC) Reference() string {
	return oa.BackendName
}

// Schema implements the <Inline> interface.
func (oa OAuth2AC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend                 *Backend `hcl:"backend,block"`
		ClientID                string   `hcl:"client_id"`
		ClientSecret            string   `hcl:"client_secret"`
		CodeVerifierValue       string   `hcl:"code_verifier_value,optional"`
		RedirectURI             string   `hcl:"redirect_uri,optional"`
		Scope                   *string  `hcl:"scope,optional"`
		TokenEndpoint           string   `hcl:"token_endpoint,optional"`
		TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}

func (oa OAuth2AC) GetGrantType() string {
	return oa.GrantType
}
