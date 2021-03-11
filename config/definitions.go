package config

import "github.com/avenga/couper/config/jwt"

// Definitions represents the <Definitions> object.
type Definitions struct {
	BasicAuth         []*BasicAuth             `hcl:"basic_auth,block"`
	JWT               []*jwt.JWT               `hcl:"jwt,block"`
	JWTSigningProfile []*jwt.JWTSigningProfile `hcl:"jwt_signing_profile,block"`
}
