package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference      = &OIDC{}
	_ BackendInitialization = &OIDC{}
	_ Inline                = &OIDC{}
)

// OIDC represents an oidc block. The backend block will be used as backend template for all
// configuration related backends. Backend references along with an anonymous one must match
// the url with the backend origin definition.
type OIDC struct {
	ErrorHandlerSetter
	BackendName             string   `hcl:"backend,optional" docs:"{backend} block reference, defined in [{definitions}](definitions). Required, if no [{backend} block](backend) or {configuration_url} is defined within."`
	ClientID                string   `hcl:"client_id" docs:"The client identifier."`
	ClientSecret            string   `hcl:"client_secret" docs:"The client password."`
	ConfigurationURL        string   `hcl:"configuration_url" docs:"The OpenID configuration URL."`
	JWKsTTL                 string   `hcl:"jwks_ttl,optional" docs:"Time period the JWK set stays valid and may be cached." type:"duration" default:"1h"`
	JWKsMaxStale            string   `hcl:"jwks_max_stale,optional" docs:"Time period the cached JWK set stays valid after its TTL has passed." type:"duration" default:"1h"`
	Name                    string   `hcl:"name,label"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional" docs:"A space separated list of requested scope values for the access token."`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional" docs:"Defines the method to authenticate the client at the token endpoint. If set to {client_secret_post}, the client credentials are transported in the request body. If set to {client_secret_basic}, the client credentials are transported via Basic Authentication." default:"client_secret_basic"`
	ConfigurationTTL        string   `hcl:"configuration_ttl,optional" docs:"The duration to cache the OpenID configuration located at {configuration_url}." type:"duration" default:"1h"`
	ConfigurationMaxStale   string   `hcl:"configuration_max_stale,optional" docs:"Duration a cached OpenID configuration stays valid after its TTL has passed." type:"duration" default:"1h"`
	VerifierMethod          string   `hcl:"verifier_method,optional" docs:"The method to verify the integrity of the authorization code flow."`

	// configuration related backends
	ConfigurationBackendName string `hcl:"configuration_backend,optional"`
	JWKSBackendName          string `hcl:"jwks_uri_backend,optional"`
	TokenBackendName         string `hcl:"token_backend,optional"`
	UserinfoBackendName      string `hcl:"userinfo_backend,optional"`

	// internally used
	Backends map[string]hcl.Body
}

func (o *OIDC) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if o.Backends == nil {
		o.Backends = make(map[string]hcl.Body)
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

// HCLBody implements the <Inline> interface.
func (o *OIDC) HCLBody() hcl.Body {
	return o.Remain
}

// Inline implements the <Inline> interface.
func (o *OIDC) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend       *Backend `hcl:"backend,block"`
		RedirectURI   string   `hcl:"redirect_uri" docs:"The Couper endpoint for receiving the authorization code. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the {accept_forwarded_url}({settings} block) attribute if Couper is running behind a proxy."`
		VerifierValue string   `hcl:"verifier_value" docs:"The value of the (unhashed) verifier."`

		AuthorizationBackend       *Backend `hcl:"authorization_backend,block"`
		ConfigurationBackend       *Backend `hcl:"configuration_backend,block" docs:"Optional option to configure specific behaviour for a given oidc backend."`
		DeviceAuthorizationBackend *Backend `hcl:"device_authorization_backend,block"`
		JWKSBackend                *Backend `hcl:"jwks_uri_backend,block" docs:"Optional option to configure specific behaviour for a given oidc backend."`
		RevocationBackend          *Backend `hcl:"revocation_backend,block"`
		TokenBackend               *Backend `hcl:"token_backend,block" docs:"Optional option to configure specific behaviour for a given oidc backend."`
		UserinfoBackend            *Backend `hcl:"userinfo_backend,block" docs:"Optional option to configure specific behaviour for a given oidc backend."`
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

	// A backend reference is defined, backend block is not allowed.
	if o.BackendName != "" {
		schema.Blocks = nil
	}

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}

func (o *OIDC) ClientAuthenticationRequired() bool {
	return true
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

func (o *OIDC) GetScope() string {
	if o.Scope == nil {
		return "openid"
	}
	return "openid " + *o.Scope
}

func (o *OIDC) GetTokenEndpointAuthMethod() *string {
	return o.TokenEndpointAuthMethod
}
