package config

// Definitions represents the <Definitions> object.
type Definitions struct {
	Backend           []*Backend           `hcl:"backend,block"`
	BasicAuth         []*BasicAuth         `hcl:"basic_auth,block"`
	JWT               []*JWT               `hcl:"jwt,block"`
	JWTSigningProfile []*JWTSigningProfile `hcl:"jwt_signing_profile,block"`
	SAML              []*SAML              `hcl:"saml,block"`
	OAuth2AC          []*OAuth2AC          `hcl:"beta_oauth2,block"`
	OIDC              []*OIDC              `hcl:"oidc,block"`
	// TODO remove for version 1.8
	BetaOIDC []*OIDC `hcl:"beta_oidc,block"`
}
