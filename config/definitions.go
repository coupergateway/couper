package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
)

var _ schema.BodySchema = Definitions{}

// Definitions represents the <Definitions> object.
type Definitions struct {
	Backend           []*Backend           `hcl:"backend,block" docs:"Configure a [backend](/configuration/block/backend) (zero or more)."`
	BasicAuth         []*BasicAuth         `hcl:"basic_auth,block" docs:"Configure a [BasicAuth access control](/configuration/block/basic_auth) (zero or more)."`
	Job               []*BetaJob           `hcl:"beta_job,block" docs:"Configure a [job](/configuration/block/job) (zero or more)."`
	JWT               []*JWT               `hcl:"jwt,block" docs:"Configure a [JWT access control](/configuration/block/jwt) (zero or more)."`
	JWTSigningProfile []*JwtSigningProfile `hcl:"jwt_signing_profile,block" docs:"Configure a [JWT signing profile](/configuration/block/jwt_signing_profile) (zero or more)."`
	SAML              []*SAML              `hcl:"saml,block" docs:"Configure a [SAML access control](/configuration/block/saml) (zero or more)."`
	OAuth2AC          []*OAuth2AC          `hcl:"beta_oauth2,block" docs:"Configure an [OAuth2 assess control](/configuration/block/beta_oauth2) (zero or more)."`
	OIDC              []*OIDC              `hcl:"oidc,block" docs:"Configure an [OIDC access control](/configuration/block/oidc) (zero or more)."`

	// used for documentation
	Proxy []*Proxy `hcl:"proxy,block" docs:"Configure a [proxy](/configuration/block/proxy) (zero or more)."`
	// PoC
	Plugins []*Plugin `hcl:"plugin,block"`
}

func (d Definitions) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(d)
	return s
}
