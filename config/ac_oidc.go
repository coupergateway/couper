package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	hclbody "github.com/avenga/couper/config/body"
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
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	ConfigurationURL        string   `hcl:"configuration_url"`
	Name                    string   `hcl:"name,label"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	ConfigurationTTL        string   `hcl:"configuration_ttl,optional"`
	VerifierMethod          string   `hcl:"verifier_method,optional"`

	// configuration related backends
	AuthorizationBackendName       string `hcl:"authorization_backend,optional"`
	ConfigurationBackendName       string `hcl:"configuration_backend,optional"`
	DeviceAuthorizationBackendName string `hcl:"device_authorization_backend,optional"`
	JWKSBackendName                string `hcl:"jwks_uri_backend,optional"`
	RevocationBackendName          string `hcl:"revocation_backend,optional"`
	TokenBackendName               string `hcl:"token_backend,optional"`
	UserinfoBackendName            string `hcl:"userinfo_backend,optional"`

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

		// exceptions
		if field == "configuration_backend" && o.ConfigurationURL != "" {
			o.Backends[field] = hclbody.MergeBodies(o.Backends[field],
				hclbody.New(hclbody.NewContentWithAttrName("_backend_url", o.ConfigurationURL)))
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
		Backend       *Backend                  `hcl:"backend,block"`
		LogFields     map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
		RedirectURI   string                    `hcl:"redirect_uri"`
		VerifierValue string                    `hcl:"verifier_value"`

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

	// A backend reference is defined, backend block is not allowed.
	if o.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, o.HCLBody())
}

func (o *OIDC) GetName() string {
	return o.Name
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
