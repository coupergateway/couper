package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var (
	_ BackendInitialization = &Introspection{}
	_ Body                  = &Introspection{}
	_ Inline                = &Introspection{}
)

type Introspection struct {
	BackendName        string             `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for introspection requests. Mutually exclusive with {backend} block."`
	ClientID           string             `hcl:"client_id" docs:"The client identifier."`
	ClientSecret       string             `hcl:"client_secret,optional" docs:"The client password. Required unless the {endpoint_auth_method} is {\"private_key_jwt\"}."`
	Endpoint           string             `hcl:"endpoint" docs:"The authorization server's {introspection_endpoint}."`
	EndpointAuthMethod *string            `hcl:"endpoint_auth_method,optional" docs:"Defines the method to authenticate the client at the introspection endpoint. If set to {\"client_secret_post\"}, the client credentials are transported in the request body. If set to {\"client_secret_basic\"}, the client credentials are transported via Basic Authentication. If set to {\"client_secret_jwt\"}, the client is authenticated via a JWT signed with the {client_secret}. If set to {\"private_key_jwt\"}, the client is authenticated via a JWT signed with its private key (see {jwt_signing_profile} block)." default:"client_secret_basic"`
	JWTSigningProfile  *JWTSigningProfile `hcl:"jwt_signing_profile,block" docs:"Configures a [JWT signing profile](/configuration/block/jwt_signing_profile) to create a client assertion if {endpoint_auth_method} is either {\"client_secret_jwt\"} or {\"private_key_jwt\"}."`
	Remain             hcl.Body           `hcl:",remain"`
	TTL                string             `hcl:"ttl" docs:"The time-to-live of a cached introspection response. With a non-positive value the introspection endpoint is called each time a token is validated." type:"duration"`

	// Internally used
	Backend    *hclsyntax.Body
	TTLSeconds int64
}

// TODO use function from config/configload
func newDiagErr(subject *hcl.Range, summary string) error {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Subject:  subject,
	}}
}

func (i *Introspection) Prepare(backendFunc PrepareBackendFunc) error {
	b, err := backendFunc("introspection_endpoint", i.Endpoint, i)
	if err != nil {
		return err
	}

	i.Backend = b

	attrs := i.Remain.(*hclsyntax.Body).Attributes
	r := attrs["ttl"].Expr.Range()

	dur, err := ParseDuration("ttl", i.TTL, -1)
	if err != nil {
		return newDiagErr(&r, err.Error())
	} else if dur == -1 {
		return newDiagErr(&r, "invalid duration")
	}

	i.TTLSeconds = int64(dur.Seconds())

	return nil
}

// Reference implements the <BackendReference> interface.
func (i *Introspection) Reference() string {
	return i.BackendName
}

// HCLBody implements the <Body> interface.
func (i *Introspection) HCLBody() *hclsyntax.Body {
	return i.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (i *Introspection) Inline() interface{} {
	type Inline struct {
		// meta.LogFieldsAttribute
		Backend *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for introspection requests (zero or one). Mutually exclusive with {backend} attribute."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (i *Introspection) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(i)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(i.Inline())

	return schema
	// return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}
