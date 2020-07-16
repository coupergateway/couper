package config

import "github.com/hashicorp/hcl/v2"

type Inline interface {
	Schema(inline bool) *hcl.BodySchema
}
