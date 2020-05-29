package backend

import (
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

type ContextOptions struct {
	RequestHeaders http.Header `hcl:"request_headers"`
}

func NewContextOptions(hclBody hcl.Body, req *http.Request) (*ContextOptions, error) {
	options := &ContextOptions{}
	decodeCtx := &hcl.EvalContext{ // TODO: maybe add parent context earlier with basics like req, env
		Variables: map[string]cty.Value{
			"env": newCtyEnvMap(),
			"req": cty.MapVal(map[string]cty.Value{
				"headers": newCtyHeadersMap(req.Header),
			}),
		},
	}
	diags := gohcl.DecodeBody(hclBody, decodeCtx, options)
	if diags.HasErrors() {
		return nil, diags
	}
	return options, nil
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
