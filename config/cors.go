package config

import (
	"github.com/zclconf/go-cty/cty"
)

// CORS represents the <CORS> object.
type CORS struct {
	AllowedOrigins   cty.Value `hcl:"allowed_origins" docs:"An allowed origin or a list of allowed origins." type:"string or tuple"`
	AllowCredentials bool      `hcl:"allow_credentials,optional" docs:"Set to {true} if the response can be shared with credentialed requests (containing {Cookie} or {Authorization} HTTP header fields)."`
	Disable          bool      `hcl:"disable,optional" docs:"Set to {true} to disable the inheritance of CORS from parent context."`
	MaxAge           string    `hcl:"max_age,optional" docs:"Indicates the time the information provided by the {Access-Control-Allow-Methods} and {Access-Control-Allow-Headers} response HTTP header fields." type:"duration"`
}
