package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	_ BackendReference    = &OAuth2AC{}
	_ Inline              = &OAuth2AC{}
	_ OAuth2AcClient      = &OAuth2AC{}
	_ OAuth2AcAS          = &OAuth2AC{}
	_ OAuth2Authorization = &OAuth2AC{}
)

// OAuth2AC represents represents an oauth2 block for an OAuth2 client using the authorization code flow.
type OAuth2AC struct {
	AccessControlSetter
	AuthorizationEndpoint   string   `hcl:"authorization_endpoint"`
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	GrantType               string   `hcl:"grant_type"`
	Name                    string   `hcl:"name,label"`
	RedirectURI             string   `hcl:"redirect_uri"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpoint           string   `hcl:"token_endpoint"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	VerifierMethod          string   `hcl:"verifier_method"`

	// internally used
	Backend     hcl.Body
	BodyContent *hcl.BodyContent
}

// Reference implements the <BackendReference> interface.
func (oa OAuth2AC) Reference() string {
	return oa.BackendName
}

// HCLBody implements the <Inline> interface.
func (oa OAuth2AC) HCLBody() hcl.Body {
	return oa.Remain
}

// Schema implements the <Inline> interface.
func (oa OAuth2AC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	type Inline struct {
		Backend       *Backend `hcl:"backend,block"`
		VerifierValue string   `hcl:"verifier_value"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, oa.HCLBody())
}

func (oa *OAuth2AC) GetBodyContent() *hcl.BodyContent {
	return oa.BodyContent
}

func (oa OAuth2AC) GetName() string {
	return oa.Name
}

func (oa OAuth2AC) GetClientID() string {
	return oa.ClientID
}

func (oa OAuth2AC) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa OAuth2AC) GetGrantType() string {
	return oa.GrantType
}

func (oa OAuth2AC) GetScope() string {
	if oa.Scope == nil {
		return ""
	}
	return *oa.Scope
}

func (oa OAuth2AC) GetRedirectURI() string {
	return oa.RedirectURI
}

func (oa OAuth2AC) GetAuthorizationEndpoint() (string, error) {
	return oa.AuthorizationEndpoint, nil
}

func (oa OAuth2AC) GetTokenEndpoint() (string, error) {
	return oa.TokenEndpoint, nil
}

func (oa OAuth2AC) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or state)
func (oa OAuth2AC) GetVerifierMethod() (string, error) {
	return oa.VerifierMethod, nil
}
