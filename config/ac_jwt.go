package config

import (
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// Internally used for 'error_handler'.
var _ Body = &JWT{}

// Claims represents the <Claims> object.
type Claims hcl.Expression

// JWT represents the <JWT> object.
type JWT struct {
	AccessControlSetter
	Claims             Claims   `hcl:"claims,optional"`
	ClaimsRequired     []string `hcl:"required_claims,optional"`
	Cookie             string   `hcl:"cookie,optional"`
	Header             string   `hcl:"header,optional"`
	JWKsURL            string   `hcl:"jwks_url,optional"`
	JWKsTTL            string   `hcl:"jwks_ttl,optional"`
	JWKSBackendRef     string   `hcl:"backend,optional"`
	Key                string   `hcl:"key,optional"`
	KeyFile            string   `hcl:"key_file,optional"`
	Name               string   `hcl:"name,label"`
	PostParam          string   `hcl:"post_param,optional"`
	QueryParam         string   `hcl:"query_param,optional"`
	ScopeClaim         string   `hcl:"beta_scope_claim,optional"`
	SignatureAlgorithm string   `hcl:"signature_algorithm,optional"`
	SigningKey         string   `hcl:"signing_key,optional"`
	SigningKeyFile     string   `hcl:"signing_key_file,optional"`
	SigningTTL         string   `hcl:"signing_ttl,optional"`

	// Internally used
	BodyContent     *hcl.BodyContent
	JWKSBackendBody hcl.Body
	Remain          hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}

func (j *JWT) Check() error {
	if j.JWKSBackendRef != "" && j.JWKSBackendBody != nil {
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
		if j.JWKSBackendRef != "" || j.JWKSBackendBody != nil {
			return errors.New("backend not needed without jwks_url")
		}

		if j.SignatureAlgorithm == "" {
			return errors.New("signature_algorithm or jwks_url required")
		}
	}

	return nil
}

// Reference implements the <BackendReference> interface.
func (j *JWT) Reference() string {
	return j.JWKSBackendRef
}

func (j *JWT) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(j)
		return schema
	}

	type Inline struct {
		Backend *Backend `hcl:"backend,block"`
	}

	schema, _ := gohcl.ImpliedBodySchema(&Inline{})

	// A backend reference is defined, backend block is not allowed.
	if j.JWKSBackendRef != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, j.HCLBody())
}
