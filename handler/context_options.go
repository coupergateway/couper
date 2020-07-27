package handler

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type RequestContext struct {
	Options MapOptions `hcl:"request_headers,optional"`
}

type ResponseContext struct {
	Options MapOptions `hcl:"response_headers,optional"`
}

type MapOptions map[string]interface{}

func NewCtxOptions(target interface{}, decodeCtx *hcl.EvalContext, body hcl.Body) error {
	diags := gohcl.DecodeBody(body, decodeCtx, target)
	if diags.HasErrors() {
		return diags
	}
	return nil
}
