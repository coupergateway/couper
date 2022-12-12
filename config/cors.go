package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/schema"
)

var _ schema.BodySchema = &CORS{}

// CORS represents the <CORS> object.
type CORS struct {
	AllowedOrigins   cty.Value `hcl:"allowed_origins" docs:"An allowed origin or a list of allowed origins."`
	AllowCredentials bool      `hcl:"allow_credentials,optional" docs:"Set to {true} if the response can be shared with credentialed requests (containing {Cookie} or {Authorization} HTTP header fields)."`
	Disable          bool      `hcl:"disable,optional" docs:"Set to {true} to disable the inheritance of CORS from parent context."`
	MaxAge           string    `hcl:"max_age,optional" docs:"Indicates the time the information provided by the {Access-Control-Allow-Methods} and {Access-Control-Allow-Headers} response HTTP header fields." type:"duration"`
}

func (c CORS) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(c)
	return s
}
