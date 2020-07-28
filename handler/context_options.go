package handler

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type ContextOptions struct {
	ReqOptions  http.Header `hcl:"request_headers,optional"`
	RespOptions http.Header `hcl:"response_headers,optional"`
}

func NewCtxOptions(target interface{}, decodeCtx *hcl.EvalContext, body hcl.Body) error {
	diags := gohcl.DecodeBody(body, decodeCtx, target)
	if diags.HasErrors() {
		return diags
	}
	return nil
}
