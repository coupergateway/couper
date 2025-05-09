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
	Name         string   `hcl:"name,label"`
	Period       string   `hcl:"period" docs:"Defines the rate limit period." type:"duration"`
	PerPeriod    int      `hcl:"per_period" docs:"Defines the number of allowed requests in a period."`
	PeriodWindow string   `hcl:"period_window,optional" default:"sliding" docs:"Defines the window of the period. A {fixed} window permits {per_period} requests within {period}. After the {period} has expired, another {per_period} request is permitted. The sliding window ensures that only {per_period} requests are sent in any interval of length {period}."`
	Remain       hcl.Body `hcl:",remain"`
}

// HCLBody implements the <Body> interface. Internally used for 'error_handler'.
func (r *RateLimiter) HCLBody() *hclsyntax.Body {
	return r.Remain.(*hclsyntax.Body)
}

func (r *RateLimiter) Inline() any {
	type Inline struct {
		meta.LogFieldsAttribute
		Key string `hcl:"key" docs:"The expression defining which key to be used to identify a visitor."`
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
