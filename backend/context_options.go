package backend

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type RequestOptions struct {
	Request  ContextOptions `hcl:"request,block"`
	Rest     hcl.Body       `hcl:",remain"`
}

type ResponseOptions struct {
	Response ContextOptions `hcl:"response,block"`
	Rest     hcl.Body       `hcl:",remain"`
}

type ContextOptions *struct {
	Headers http.Header `hcl:"headers,optional"`
}

func NewRequestCtxOptions(hclBody hcl.Body, req *http.Request) (*RequestOptions, error) {
	decodeCtx := NewEvalContext(req, nil)

	options := &RequestOptions{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}

func NewResponseCtxOptions(hclBody hcl.Body, res *http.Response) (*ResponseOptions, error) {
	decodeCtx := NewEvalContext(res.Request, res)

	options := &ResponseOptions{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}
