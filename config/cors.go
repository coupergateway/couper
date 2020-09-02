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

func (c *CORS) NeedsVary() bool {
	// If request with not allowed Origin is ignored
	return !c.AllowsOrigin("*")
	// Otherwise
	// return len(seetie.ValueToStringSlice(c.AllowedOrigins)) > 1
}

func (c* CORS) AllowsOrigin(origin string) bool {
	for _, a := range seetie.ValueToStringSlice(c.AllowedOrigins) {
		if a == origin || a == "*" {
			return true
		}
	}
	return false
}
