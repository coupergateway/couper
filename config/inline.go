package config

import "github.com/hashicorp/hcl/v2"

type Inline interface {
	Body() hcl.Body
	Reference() string
	Schema(inline bool) *hcl.BodySchema
}
