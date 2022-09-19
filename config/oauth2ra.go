package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const (
	ClientCredentials = "client_credentials"
	JwtBearer         = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	Password          = "password"
)

var oauthBlockHeaderSchema = hcl.BlockHeaderSchema{
	Type: "oauth2",
}
var OAuthBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		oauthBlockHeaderSchema,
	},
}

var (
	_ BackendReference = &OAuth2ReqAuth{}
	_ Body             = &OAuth2ReqAuth{}
	_ Inline           = &OAuth2ReqAuth{}
	_ OAuth2Client     = &OAuth2ReqAuth{}
	_ OAuth2AS         = &OAuth2ReqAuth{}
)

// OAuth2ReqAuth represents the oauth2 block in a backend block.
type OAuth2ReqAuth struct {
	AssertionExpr           hcl.Expression `hcl:"assertion,optional" docs:"The assertion (JWT for jwt-bearer flow). Required if {grant_type} is {urn:ietf:params:oauth:grant-type:jwt-bearer}." type:"string"`
	AuthnAudClaim           string         `hcl:"authn_aud_claim,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The {aud} claim value. Default: The value of {token_endpoint}."`
	AuthnKey                string         `hcl:"authn_key,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: The private key to sign the token."`
	AuthnKeyFile            string         `hcl:"authn_key_file,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: Optional file reference instead of {authn_key} usage."`
	AuthnSignatureAlgotithm string         `hcl:"authn_signature_algorithm,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The algorithm to use for signing the token: {\"HS256\"}, {\"HS384\"} or {\"HS512\"} for {\"client_secret_jwt\"}, {\"RS256\"}, {\"RS384\"}, {\"RS512\"}, {\"ES256\"}, {\"ES384\"} or {\"ES512\"} for {\"private_key_jwt\"}."`
	AuthnTTL                string         `hcl:"authn_ttl,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The token's time-to-live (creates the {exp} claim)." type:"duration"`
	BackendName             string         `hcl:"backend,optional" docs:"[{backend} block](backend) reference."`
	ClientID                string         `hcl:"client_id,optional" docs:"The client identifier. Required unless the {grant_type} is {urn:ietf:params:oauth:grant-type:jwt-bearer}."`
	ClientSecret            string         `hcl:"client_secret,optional" docs:"The client password. Required unless the {grant_type} is {urn:ietf:params:oauth:grant-type:jwt-bearer}."`
	GrantType               string         `hcl:"grant_type" docs:"Required, valid values: {client_credentials}, {password}, {urn:ietf:params:oauth:grant-type:jwt-bearer}"`
	Password                string         `hcl:"password,optional" docs:"The (service account's) password (for password flow). Required if grant_type is {password}."`
	Remain                  hcl.Body       `hcl:",remain"`
	Retries                 *uint8         `hcl:"retries,optional" default:"1" docs:"The number of retries to get the token and resource, if the resource-request responds with {401 Unauthorized} HTTP status code."`
	Scope                   string         `hcl:"scope,optional" docs:"A space separated list of requested scope values for the access token."`
	TokenEndpoint           string         `hcl:"token_endpoint,optional" docs:"URL of the token endpoint at the authorization server."`
	TokenEndpointAuthMethod *string        `hcl:"token_endpoint_auth_method,optional" docs:"Defines the method to authenticate the client at the token endpoint. If set to {\"client_secret_post\"}, the client credentials are transported in the request body. If set to {\"client_secret_basic\"}, the client credentials are transported via Basic Authentication. If set to {\"client_secret_jwt\"}, the client is authenticated via a JWT signed with the {client_secret}. If set to {\"private_key_jwt\"}, the client is authenticated via a JWT signed with its private key (see {authn_key} or {authn_key_file})." default:"client_secret_basic"`
	Username                string         `hcl:"username,optional" docs:"The (service account's) username (for password flow). Required if grant_type is {password}."`
}

// Reference implements the <BackendReference> interface.
func (oa *OAuth2ReqAuth) Reference() string {
	return oa.BackendName
}

// HCLBody implements the <Body> interface.
func (oa *OAuth2ReqAuth) HCLBody() *hclsyntax.Body {
	return oa.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (oa *OAuth2ReqAuth) Inline() interface{} {
	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (oa *OAuth2ReqAuth) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(oa.Inline())

	return schema
}

func (oa *OAuth2ReqAuth) ClientAuthenticationRequired() bool {
	return oa.GrantType != JwtBearer
}

func (oa *OAuth2ReqAuth) GetClientID() string {
	return oa.ClientID
}

func (oa *OAuth2ReqAuth) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa *OAuth2ReqAuth) GetTokenEndpoint() (string, error) {
	return oa.TokenEndpoint, nil
}

func (oa *OAuth2ReqAuth) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}
