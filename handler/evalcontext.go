package handler

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func NewEvalContext(request *http.Request, response *http.Response) *hcl.EvalContext {
	variables := make(map[string]cty.Value)

	variables["env"] = newCtyEnvMap()

	if request != nil {
		variables["req"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(request.Header),
			"cookies": newCtyCookiesMap(request),
			//"params":  newCtyParametersMap(mux.Vars(request)),
		})
	}

	if response != nil {
		variables["res"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(response.Header),
		})
	}

	return &hcl.EvalContext{
		Variables: variables,
		Functions: newFunctionsMap(),
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
		"to_lower": toLower(), // Custom function
	}
}

// Example function
func toLower() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "s",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			s := args[0].AsString()
			return cty.StringVal(strings.ToLower(s)), nil
		},
	})
}
