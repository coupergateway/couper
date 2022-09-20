package config

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference      = &OIDC{}
	_ BackendInitialization = &OIDC{}
	_ Body                  = &OIDC{}
	_ Inline                = &OIDC{}
)

// OIDC represents an oidc block. The backend block will be used as backend template for all
// configuration related backends. Backend references along with an anonymous one must match
// the url with the backend origin definition.
type OIDC struct {
	ErrorHandlerSetter
	AuthnAudClaim           string   `hcl:"authn_aud_claim,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The {aud} claim value. Default: The value of {token_endpoint}."`
	AuthnKey                string   `hcl:"authn_key,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: The private key to sign the token."`
	AuthnKeyFile            string   `hcl:"authn_key_file,optional" docs:"For {token_endpoint_auth_method} value {\"private_key_jwt\"}: Optional file reference instead of {authn_key} usage."`
	AuthnSignatureAlgotithm string   `hcl:"authn_signature_algorithm,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The algorithm to use for signing the token: {\"HS256\"}, {\"HS384\"} or {\"HS512\"} for {\"client_secret_jwt\"}, {\"RS256\"}, {\"RS384\"}, {\"RS512\"}, {\"ES256\"}, {\"ES384\"} or {\"ES512\"} for {\"private_key_jwt\"}."`
	AuthnTTL                string   `hcl:"authn_ttl,optional" docs:"For {token_endpoint_auth_method} values {\"client_secret_jwt\"} or {\"private_key_jwt\"}: The token's time-to-live (creates the {exp} claim)." type:"duration"`
	AuthnX5tHeader          string   `hcl:"authn_x5t_header,optional" docs:"For {token_endpoint_auth_method} values {\"private_key_jwt\"}: The {x5t} header value."`
	BackendName             string   `hcl:"backend,optional" docs:"{backend} block reference, defined in [{definitions}](definitions). Default for OpenID configuration, JWKS, token and userinfo requests."`
	ClientID                string   `hcl:"client_id" docs:"The client identifier."`
	ClientSecret            string   `hcl:"client_secret,optional" docs:"The client password."`
	ConfigurationURL        string   `hcl:"configuration_url" docs:"The OpenID configuration URL."`
	JWKsTTL                 string   `hcl:"jwks_ttl,optional" docs:"Time period the JWK set stays valid and may be cached." type:"duration" default:"1h"`
	JWKsMaxStale            string   `hcl:"jwks_max_stale,optional" docs:"Time period the cached JWK set stays valid after its TTL has passed." type:"duration" default:"1h"`
	Name                    string   `hcl:"name,label"`
	Remain                  hcl.Body `hcl:",remain"`
	RedirectURI             string   `hcl:"redirect_uri" docs:"The Couper endpoint for receiving the authorization code. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [{accept_forwarded_url} attribute](settings) if Couper is running behind a proxy."`
	Scope                   string   `hcl:"scope,optional" docs:"A space separated list of requested scope values for the access token."`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional" docs:"Defines the method to authenticate the client at the token endpoint. If set to {\"client_secret_post\"}, the client credentials are transported in the request body. If set to {\"client_secret_basic\"}, the client credentials are transported via Basic Authentication. If set to {\"client_secret_jwt\"}, the client is authenticated via a JWT signed with the {client_secret}. If set to {\"private_key_jwt\"}, the client is authenticated via a JWT signed with its private key (see {authn_key} or {authn_key_file})." default:"client_secret_basic"`
	ConfigurationTTL        string   `hcl:"configuration_ttl,optional" docs:"The duration to cache the OpenID configuration located at {configuration_url}." type:"duration" default:"1h"`
	ConfigurationMaxStale   string   `hcl:"configuration_max_stale,optional" docs:"Duration a cached OpenID configuration stays valid after its TTL has passed." type:"duration" default:"1h"`
	VerifierMethod          string   `hcl:"verifier_method,optional" docs:"The method to verify the integrity of the authorization code flow."`

	// configuration related backends
	ConfigurationBackendName string `hcl:"configuration_backend,optional" docs:"Optional option to configure specific behavior for the backend to request the OpenID configuration from."`
	JWKSBackendName          string `hcl:"jwks_uri_backend,optional" docs:"Optional option to configure specific behavior for the backend to request the JWKS from."`
	TokenBackendName         string `hcl:"token_backend,optional" docs:"Optional option to configure specific behavior for the backend to request the token from."`
	UserinfoBackendName      string `hcl:"userinfo_backend,optional" docs:"Optional option to configure specific behavior for the backend to request the userinfo from."`

	// internally used
	Backends map[string]*hclsyntax.Body
}

func (o *OIDC) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if o.Backends == nil {
		o.Backends = make(map[string]*hclsyntax.Body)
	}

	fields := BackendAttrFields(o)
	for _, field := range fields {
		fieldValue := AttrValueFromTagField(field, o)
		o.Backends[field], err = backendFunc(field, fieldValue, o)
		if err != nil {
			return err
		}
	}
	return nil
}

// Reference implements the <BackendReference> interface.
func (o *OIDC) Reference() string {
	return o.BackendName
}

// HCLBody implements the <Body> interface.
func (o *OIDC) HCLBody() *hclsyntax.Body {
	return o.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (o *OIDC) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend       *Backend `hcl:"backend,block"`
		VerifierValue string   `hcl:"verifier_value" docs:"The value of the (unhashed) verifier."`

		AuthorizationBackend       *Backend `hcl:"authorization_backend,block"`
		ConfigurationBackend       *Backend `hcl:"configuration_backend,block"`
		DeviceAuthorizationBackend *Backend `hcl:"device_authorization_backend,block"`
		JWKSBackend                *Backend `hcl:"jwks_uri_backend,block"`
		RevocationBackend          *Backend `hcl:"revocation_backend,block"`
		TokenBackend               *Backend `hcl:"token_backend,block"`
		UserinfoBackend            *Backend `hcl:"userinfo_backend,block"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (o *OIDC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(o)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(o.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}

func (o *OIDC) ClientAuthenticationRequired() bool {
	return true
}

func (o *OIDC) GetAuthnAudClaim() string {
	return o.AuthnAudClaim
}

func (o *OIDC) GetAuthnKey() string {
	return o.AuthnKey
}

func (o *OIDC) GetAuthnKeyFile() string {
	return o.AuthnKeyFile
}

func (o *OIDC) GetAuthnSignatureAlgotithm() string {
	return o.AuthnSignatureAlgotithm
}

func (o *OIDC) GetAuthnTTL() string {
	return o.AuthnTTL
}

func (o *OIDC) GetAuthnX5tHeader() string {
	return o.AuthnX5tHeader
}

func (o *OIDC) GetClientID() string {
	return o.ClientID
}

func (o *OIDC) GetClientSecret() string {
	return o.ClientSecret
}

func (o *OIDC) GetGrantType() string {
	return "authorization_code"
}

func (o *OIDC) GetRedirectURI() string {
	return o.RedirectURI
}

func (o *OIDC) GetScope() string {
	scope := strings.TrimSpace(o.Scope)
	if scope == "" {
		return "openid"
	}

	return "openid " + scope
}

func (o *OIDC) GetTokenEndpointAuthMethod() *string {
	return o.TokenEndpointAuthMethod
}
