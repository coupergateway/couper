package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/meta"
)

var (
	_ Body   = &RateLimiter{}
	_ Inline = &RateLimiter{}
)

// RateLimiter represents the "beta_rate_limiter" config block
type RateLimiter struct {
	ErrorHandlerSetter
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (r *RateLimiter) HCLBody() *hclsyntax.Body {
	return r.Remain.(*hclsyntax.Body)
}

func (r *RateLimiter) Inline() any {
	type Inline struct {
		meta.LogFieldsAttribute
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (r *RateLimiter) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(r)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(r.Inline())
	return schema
}
