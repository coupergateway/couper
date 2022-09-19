package config

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference      = &OAuth2AC{}
	_ BackendInitialization = &OAuth2AC{}
	_ Inline                = &OAuth2AC{}
	_ OAuth2AS              = &OAuth2AC{}
	_ OAuth2AcClient        = &OAuth2AC{}
	_ OAuth2Authorization   = &OAuth2AC{}
)

// OAuth2AC represents an oauth2 block for an OAuth2 client using the authorization code flow.
type OAuth2AC struct {
	ErrorHandlerSetter
	AuthnAudClaim           string `hcl:"authn_aud_claim,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The {aud} claim value. Default: The value of {token_endpoint}."`
	AuthnKey                string `hcl:"authn_key,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: The private key to sign the token."`
	AuthnKeyFile            string `hcl:"authn_key_file,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: Optional file reference instead of {authn_key} usage."`
	AuthnSignatureAlgotithm string `hcl:"authn_signature_algorithm,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The algorithm to use for signing the token: {\"HS256\"}, {\"HS384\"} or {\"HS512\"} for {\"client_secret_jwt\"}, {\"RS256\"}, {\"RS384\"}, {\"RS512\"}, {\"ES256\"}, {\"ES384\"} or {\"ES512\"} for {\"private_key_jwt\"}."`
	AuthnTTL                string `hcl:"authn_ttl,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The token's time-to-live (creates the {exp} claim)." type:"duration"`
	// AuthorizationEndpoint is used for lib.FnOAuthAuthorizationURL
	AuthorizationEndpoint   string   `hcl:"authorization_endpoint" docs:"The authorization server endpoint URL used for authorization."`
	BackendName             string   `hcl:"backend,optional" docs:"[{backend} block](backend) reference."`
	ClientID                string   `hcl:"client_id" docs:"The client identifier."`
	ClientSecret            string   `hcl:"client_secret" docs:"The client password."`
	GrantType               string   `hcl:"grant_type" docs:"The grant type. Required, to be set to: {authorization_code}"`
	Name                    string   `hcl:"name,label"`
	RedirectURI             string   `hcl:"redirect_uri" docs:"The Couper endpoint for receiving the authorization code. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [{accept_forwarded_url} attribute](settings) if Couper is running behind a proxy."`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   string   `hcl:"scope,optional" docs:"A space separated list of requested scope values for the access token."`
	TokenEndpoint           string   `hcl:"token_endpoint" docs:"The authorization server endpoint URL used for requesting the token."`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional" docs:"Defines the method to authenticate the client at the token endpoint. If set to {\"client_secret_post\"}, the client credentials are transported in the request body. If set to {\"client_secret_basic\"}, the client credentials are transported via Basic Authentication. If set to {\"client_secret_jwt\"}, the client is authenticated via a JWT signed with the {client_secret}. If set to {\"private_key_jwt\"}, the client is authenticated via a JWT signed with its private key (see {authn_key} or {authn_key_file})." default:"client_secret_basic"`
	VerifierMethod          string   `hcl:"verifier_method" docs:"The method to verify the integrity of the authorization code flow. Available values: {ccm_s256} ({code_challenge} parameter with {code_challenge_method} {S256}), {state} ({state} parameter)"`

	// internally used
	Backend *hclsyntax.Body
}

func (oa *OAuth2AC) Prepare(backendFunc PrepareBackendFunc) (err error) {
	oa.Backend, err = backendFunc("token_endpoint", oa.TokenEndpoint, oa)
	return err
}

// Reference implements the <BackendReference> interface.
func (oa *OAuth2AC) Reference() string {
	return oa.BackendName
}

// HCLBody implements the <Body> interface.
func (oa *OAuth2AC) HCLBody() *hclsyntax.Body {
	return oa.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (oa *OAuth2AC) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend       *Backend `hcl:"backend,block"`
		VerifierValue string   `hcl:"verifier_value" docs:"The value of the (unhashed) verifier. E.g. using cookie value created with {oauth2_verifier()} function](../functions)"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (oa *OAuth2AC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(oa.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}

func (oa *OAuth2AC) ClientAuthenticationRequired() bool {
	return true
}

func (oa *OAuth2AC) GetClientID() string {
	return oa.ClientID
}

func (oa *OAuth2AC) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa *OAuth2AC) GetGrantType() string {
	return oa.GrantType
}

func (oa *OAuth2AC) GetRedirectURI() string {
	return oa.RedirectURI
}

func (oa *OAuth2AC) GetScope() string {
	return strings.TrimSpace(oa.Scope)
}

func (oa *OAuth2AC) GetAuthorizationEndpoint() (string, error) {
	return oa.AuthorizationEndpoint, nil
}

func (oa *OAuth2AC) GetTokenEndpoint() (string, error) {
	return oa.TokenEndpoint, nil
}

func (oa *OAuth2AC) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or state)
func (oa *OAuth2AC) GetVerifierMethod() (string, error) {
	return oa.VerifierMethod, nil
}
