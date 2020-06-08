package backend

import (
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type Options struct {
	Request  ContextOptions `hcl:"request,block"`
	Response ContextOptions `hcl:"response,block"`
}

type ContextOptions struct {
	Headers http.Header `hcl:"headers,optional"`
}

func NewRequestCtxOptions(hclBody hcl.Body, req *http.Request) (*Options, error) {
	decodeCtx := newEvalContext()
	decodeCtx.Variables["req"] = cty.MapVal(map[string]cty.Value{
		"headers": newCtyHeadersMap(req.Header),
	})

	options := &Options{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}

func NewResponseCtxOptions(hclBody hcl.Body, res *http.Response) (*Options, error) {
	decodeCtx := newEvalContext()
	decodeCtx.Variables["req"] = cty.MapVal(map[string]cty.Value{
		"headers": newCtyHeadersMap(res.Request.Header),
	})
	decodeCtx.Variables["res"] = cty.MapVal(map[string]cty.Value{
		"headers": newCtyHeadersMap(res.Header),
	})

	options := &Options{}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
}

func newEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"env": newCtyEnvMap(),
		},
		Functions: map[string]function.Function{
			"to_upper": stdlib.UpperFunc,
			"to_lower": to_lower(), // Custom function
		},
	}
}

func newCtyEnvMap() cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, v := range os.Environ() {
		kv := strings.Split(v, "=") // TODO: multiple vals
		if _, ok := ctyMap[kv[0]]; !ok {
			ctyMap[kv[0]] = cty.StringVal(kv[1])
		}
	}
	return cty.MapVal(ctyMap)
}

func newCtyHeadersMap(headers http.Header) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range headers {
		ctyMap[k] = cty.StringVal(v[0]) // TODO: ListVal??
	}
	return cty.MapVal(ctyMap)
}

// Example function
func to_lower() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "s",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			s := cty.Value(args[0]).AsString()
			return cty.StringVal(strings.ToLower(s)), nil
		},
	})
}
