package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/schema"
)

var _ schema.BodySchema = &OpenAPI{}

// OpenAPI represents the <OpenAPI> object.
type OpenAPI struct {
	File                     string `hcl:"file" docs:"OpenAPI YAML definition file."`
	IgnoreRequestViolations  bool   `hcl:"ignore_request_violations,optional" docs:"Logs request validation results, skips error handling."`
	IgnoreResponseViolations bool   `hcl:"ignore_response_violations,optional" docs:"Logs response validation results, skips error handling."`
}

func (o *OpenAPI) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(o)
	return s
}
