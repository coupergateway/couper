package config

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/internal/seetie"
)

type CORS struct {
	AllowedOrigins   cty.Value `hcl:"allowed_origins"`	// TODO auch string erlauben
	AllowCredentials bool      `hcl:"allow_credentials,optional"`
	MaxAge           string    `hcl:"max_age,optional"`
}

func (c* CORS) AllowsOrigin(origin string) bool {
	for _, a := range seetie.ValueToStringSlice(c.AllowedOrigins) {
		if a == origin || a == "*" {
			return true
		}
	}
	return false
}
