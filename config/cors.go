package config

import (
	"github.com/zclconf/go-cty/cty"
)

// CORS represents the <CORS> object.
type CORS struct {
	AllowedOrigins   cty.Value `hcl:"allowed_origins" docs:"An allowed origin or a list of allowed origins."`
	AllowCredentials bool      `hcl:"allow_credentials,optional" docs:"Set to <code>true</code> if the response can be shared with credentialed requests (containing <code>Cookie</code> or <code>Authorization</code> HTTP header fields)."`
	Disable          bool      `hcl:"disable,optional" docs:"Set to <code>true</code> to disable the inheritance of CORS from parent context."`
	MaxAge           string    `hcl:"max_age,optional" docs:"Indicates the time the information provided by the <code>Access-Control-Allow-Methods</code> and <code>Access-Control-Allow-Headers</code> response HTTP header fields." type:"duration"`
}
