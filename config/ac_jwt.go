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
	JWKsURI            string   `hcl:"jwks_uri,optional"`
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
	JWKSBackendBody    hcl.Body

	// Internally used for 'error_handler'.
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}

func (j *JWT) ParseInlineBackend(evalContext *hcl.EvalContext) error {
	type inlineBackend struct {
		Backend *Backend `hcl:"backend,block"`
	}

	body := j.HCLBody()
	schema, _ := gohcl.ImpliedBodySchema(&inlineBackend{})
	backendSchema := newBackendSchema(schema, body)
	content, _, diags := body.PartialContent(backendSchema)
	if diags.HasErrors() {
		return diags
	}

	inlineBackends := content.Blocks.OfType("backend")
	var err error
	if len(inlineBackends) > 0 {
		j.JWKSBackendBody = inlineBackends[0].Body
	}

	return err
}

func (j *JWT) Check() error {
	if j.JWKSBackendRef != "" && j.JWKSBackendBody != nil {
		return errors.New("backend must be either block or attribute")
	}

	if j.JWKsURI != "" {
		attributes := map[string]string{
			"signature_algorithm": j.SignatureAlgorithm,
			"key_file":            j.KeyFile,
			"key":                 j.Key,
		}

		for name, value := range attributes {
			if value != "" {
				return errors.New(name + " cannot be used together with jwks_uri")
			}
		}
	} else {
		if j.JWKSBackendRef != "" || j.JWKSBackendBody != nil {
			return errors.New("backend requires jwks_uri")
		}

		if j.SignatureAlgorithm == "" {
			return errors.New("signature_algorithm or jwks_uri required")
		}
	}

	return nil
}
