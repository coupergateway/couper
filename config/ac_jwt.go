package config

import (
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &JWT{}

// Claims represents the <Claims> object.
type Claims hcl.Expression

// JWT represents the <JWT> object.
type JWT struct {
	ErrorHandlerSetter
	BackendName        string              `hcl:"backend,optional"`
	Claims             Claims              `hcl:"claims,optional"`
	ClaimsRequired     []string            `hcl:"required_claims,optional"`
	Cookie             string              `hcl:"cookie,optional"`
	Header             string              `hcl:"header,optional"`
	JWKsURL            string              `hcl:"jwks_url,optional"`
	JWKsTTL            string              `hcl:"jwks_ttl,optional"`
	Key                string              `hcl:"key,optional"`
	KeyFile            string              `hcl:"key_file,optional"`
	Name               string              `hcl:"name,label"`
	Remain             hcl.Body            `hcl:",remain"`
	RolesClaim         string              `hcl:"beta_roles_claim,optional"`
	RolesMap           map[string][]string `hcl:"beta_roles_map,optional"`
	ScopeClaim         string              `hcl:"beta_scope_claim,optional"`
	SignatureAlgorithm string              `hcl:"signature_algorithm,optional"`
	SigningKey         string              `hcl:"signing_key,optional"`
	SigningKeyFile     string              `hcl:"signing_key_file,optional"`
	SigningTTL         string              `hcl:"signing_ttl,optional"`
	TokenValue         hcl.Expression      `hcl:"token_value,optional"`

	// Internally used
	BodyContent *hcl.BodyContent
	Backend     hcl.Body
}

// Reference implements the <BackendReference> interface.
func (j *JWT) Reference() string {
	return j.BackendName
}

// HCLBody implements the <Body> interface.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}

// Inline implements the <Inline> interface.
func (j *JWT) Inline() interface{} {
	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
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

	// A backend reference is defined, backend block is not allowed.
	if j.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, j.HCLBody())
}

func (j *JWT) Check() error {
	if j.BackendName != "" && j.Backend != nil {
		return errors.New("backend must be either block or attribute")
	}

	if j.JWKsURL != "" {
		attributes := map[string]string{
			"signature_algorithm": j.SignatureAlgorithm,
			"key_file":            j.KeyFile,
			"key":                 j.Key,
		}

		for name, value := range attributes {
			if value != "" {
				return errors.New(name + " cannot be used together with jwks_url")
			}
		}
	} else {
		if j.BackendName != "" || j.Backend != nil {
			return errors.New("backend not needed without jwks_url")
		}

		if j.SignatureAlgorithm == "" {
			return errors.New("signature_algorithm or jwks_url required")
		}
	}

	return nil
}
