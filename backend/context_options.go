package backend

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type Options struct {
	Request  ContextOptions `hcl:"request,block"`
	Response ContextOptions `hcl:"response,block"`
}

type ContextOptions *struct {
	Headers http.Header `hcl:"headers,optional"`
}

func NewRequestCtxOptions(hclBody hcl.Body, req *http.Request) (*Options, error) {
	decodeCtx := NewEvalContext(req, nil)

	options := &Options{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}

func NewResponseCtxOptions(hclBody hcl.Body, res *http.Response) (*Options, error) {
	decodeCtx := NewEvalContext(res.Request, res)

	options := &Options{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}
