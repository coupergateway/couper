package config

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config/meta"
	"github.com/avenga/couper/errors"
)

var (
	_ BackendInitialization = &JWT{}
	_ Body                  = &JWT{}
	_ Inline                = &JWT{}
)

// Claims represents the <Claims> object.
type Claims hcl.Expression

// JWT represents the <JWT> object.
type JWT struct {
	ErrorHandlerSetter
	BackendName           string              `hcl:"backend,optional" docs:"References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for JWKS requests. Mutually exclusive with {backend} block."`
	Claims                Claims              `hcl:"claims,optional" docs:"Object with claims that must be given for a valid token (equals comparison with JWT payload). The claim values are evaluated per request."`
	ClaimsRequired        []string            `hcl:"required_claims,optional" docs:"List of claim names that must be given for a valid token."`
	Cookie                string              `hcl:"cookie,optional" docs:"Read token value from a cookie. Cannot be used together with {header} or {token_value}"`
	DisablePrivateCaching bool                `hcl:"disable_private_caching,optional" docs:"If set to {true}, Couper does not add the {private} directive to the {Cache-Control} HTTP header field value."`
	Header                string              `hcl:"header,optional" docs:"Read token value from the given request header field. Implies {Bearer} if {Authorization} (case-insensitive) is used, otherwise any other header name can be used. Cannot be used together with {cookie} or {token_value}."`
	JWKsURL               string              `hcl:"jwks_url,optional" docs:"URI pointing to a set of [JSON Web Keys (RFC 7517)](https://datatracker.ietf.org/doc/html/rfc7517)"`
	JWKsTTL               string              `hcl:"jwks_ttl,optional" docs:"Time period the JWK set stays valid and may be cached." type:"duration" default:"1h"`
	JWKsMaxStale          string              `hcl:"jwks_max_stale,optional" docs:"Time period the cached JWK set stays valid after its TTL has passed." type:"duration" default:"1h"`
	Key                   string              `hcl:"key,optional" docs:"Public key (in PEM format) for {RS*} and {ES*} variants or the secret for {HS*} algorithm. Mutually exclusive with {key_file}."`
	KeyFile               string              `hcl:"key_file,optional" docs:"Reference to file containing verification key. Mutually exclusive with {key}. See {key} for more information."`
	Name                  string              `hcl:"name,label"`
	Remain                hcl.Body            `hcl:",remain"`
	RolesClaim            string              `hcl:"beta_roles_claim,optional" docs:"Name of claim specifying the roles of the user represented by the token. The claim value must either be a string containing a space-separated list of role values or a list of string role values."`
	RolesMap              map[string][]string `hcl:"beta_roles_map,optional" docs:"Mapping of roles to granted permissions. Non-mapped roles can be assigned with {*} to specific permissions. Mutually exclusive with {beta_roles_map_file}."`
	RolesMapFile          string              `hcl:"beta_roles_map_file,optional" docs:"Reference to JSON file containing role mappings. Mutually exclusive with {beta_roles_map}. See {beta_roles_map} for more information."`
	PermissionsClaim      string              `hcl:"beta_permissions_claim,optional" docs:"Name of claim containing the granted permissions. The claim value must either be a string containing a space-separated list of permissions or a list of string permissions."`
	PermissionsMap        map[string][]string `hcl:"beta_permissions_map,optional" docs:"Mapping of granted permissions to additional granted permissions. Maps values from {beta_permissions_claim} and those created from {beta_roles_map}. The map is called recursively. Mutually exclusive with {beta_permissions_map_file}."`
	PermissionsMapFile    string              `hcl:"beta_permissions_map_file,optional" docs:"Reference to JSON file containing permission mappings. Mutually exclusive with {beta_permissions_map}. See {beta_permissions_map} for more information."`
	SignatureAlgorithm    string              `hcl:"signature_algorithm,optional" docs:"Valid values: {RS256}, {RS384}, {RS512}, {HS256}, {HS384}, {HS512}, {ES256}, {ES384}, {ES512}"`
	SigningKey            string              `hcl:"signing_key,optional" docs:"Private key (in PEM format) for {RS*} and {ES*} variants. Mutually exclusive with {signing_key_file}."`
	SigningKeyFile        string              `hcl:"signing_key_file,optional" docs:"Reference to file containing signing key. Mutually exclusive with {signing_key}. See {signing_key} for more information."`
	SigningTTL            string              `hcl:"signing_ttl,optional" docs:"The token's time-to-live (creates the {exp} claim)." type:"duration"`
	TokenValue            hcl.Expression      `hcl:"token_value,optional" docs:"Expression to obtain the token. Cannot be used together with {cookie} or {header}." type:"string"`

	// Internally used
	Backend *hclsyntax.Body
}

func (j *JWT) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if j.JWKsURL != "" {
		j.Backend, err = backendFunc("jwks_url", j.JWKsURL, j)
		if err != nil {
			return err
		}
	}

	if err = j.check(); err != nil {
		return errors.Configuration.Label(j.Name).With(err)
	}
	return nil
}

// Reference implements the <BackendReference> interface.
func (j *JWT) Reference() string {
	return j.BackendName
}

// HCLBody implements the <Body> interface.
func (j *JWT) HCLBody() *hclsyntax.Body {
	return j.Remain.(*hclsyntax.Body)
}

// Inline implements the <Inline> interface.
func (j *JWT) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend *Backend `hcl:"backend,block" docs:"Configures a [backend](/configuration/block/backend) for JWKS requests. Mutually exclusive with {backend} attribute."`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (j *JWT) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(j)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(j.Inline())

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}

func (j *JWT) check() error {
	if j.JWKsURL == "" && j.SignatureAlgorithm == "" {
		return fmt.Errorf("signature_algorithm or jwks_url attribute required")
	}

	if j.JWKsURL == "" && (j.BackendName != "" || j.Backend != nil) {
		return fmt.Errorf("backend is obsolete without jwks_url attribute")
	} else if j.BackendName != "" && j.Backend == nil {
		return fmt.Errorf("backend must be either a block or an attribute")
	}

	if j.JWKsURL != "" {
		attributes := map[string]string{
			"signature_algorithm": j.SignatureAlgorithm,
			"key_file":            j.KeyFile,
			"key":                 j.Key,
		}

		for name, value := range attributes {
			if value != "" {
				return fmt.Errorf("%s cannot be used together with jwks_url", name)
			}
		}
	}

	return nil
}
