package handler

import (
	"net/http"
	"os"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func NewEvalContext(envKeys []string) *hcl.EvalContext {
	variables := make(map[string]cty.Value)
	variables["env"] = newCtyEnvMap(envKeys)

	return &hcl.EvalContext{
		Variables: variables,
		Functions: newFunctionsMap(),
	}
}

func NewHTTPEvalContext(parent *hcl.EvalContext, request *http.Request, response *http.Response) *hcl.EvalContext {
	ctx := parent.NewChild()

	if request != nil {
		ctx.Variables["req"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(request.Header),
			"cookies": newCtyCookiesMap(request),
			//"params":  newCtyParametersMap(mux.Vars(request)),
		})
	}

	if response != nil {
		ctx.Variables["res"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(response.Header),
		})
	}

	return ctx
}

func newCtyEnvMap(envKeys []string) cty.Value {
	if len(envKeys) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	ctyMap := make(map[string]cty.Value)
	for _, key := range envKeys {
		if _, ok := ctyMap[key]; !ok {
			ctyMap[key] = cty.StringVal(os.Getenv(key))
		}
	}
	return cty.MapVal(ctyMap)
}

func newCtyHeadersMap(headers http.Header) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range headers {
		if isValidKey(k) {
			ctyMap[k] = cty.StringVal(v[0]) // TODO: ListVal??
		}
	}
	return cty.MapVal(ctyMap)
}

func newCtyCookiesMap(req *http.Request) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, cookie := range req.Cookies() {
		ctyMap[cookie.Name] = cty.StringVal(cookie.Value) // TODO: ListVal??
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func newCtyParametersMap(parameters map[string]string) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range parameters {
		if isValidKey(k) {
			ctyMap[k] = cty.StringVal(v)
		}
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func isValidKey(key string) bool {
	valid, _ := regexp.MatchString("[a-zA-Z_][a-zA-Z0-9_-]*", key)
	return valid
}

// Functions
func newFunctionsMap() map[string]function.Function {
	return map[string]function.Function{
		"to_upper": stdlib.UpperFunc,
		"to_lower": stdlib.LowerFunc,
	}
}
