package config

import (
	"github.com/zclconf/go-cty/cty"
)

type CORS struct {
	AllowedOrigins   cty.Value `hcl:"allowed_origins"`
	AllowCredentials bool      `hcl:"allow_credentials,optional"`
	MaxAge           string    `hcl:"max_age,optional"`
}
