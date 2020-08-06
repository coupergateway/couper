package eval

import (
	"bytes"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func NewENVContext(src []byte) *hcl.EvalContext {
	envKeys := decodeEnvironmentRefs(src)
	variables := make(map[string]cty.Value)
	variables["env"] = newCtyEnvMap(envKeys)

	return &hcl.EvalContext{
		Variables: variables,
		Functions: newFunctionsMap(),
	}
}

func NewHTTPContext(ctx *hcl.EvalContext, req *http.Request, beresp *http.Response) *hcl.EvalContext {
	if req != nil {
		ctx.Variables["req"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(req.Header),
			"cookies": newCtyCookiesMap(req.Cookies()),
			//"params":  newCtyParametersMap(mux.Vars(request)),
		})
		if req.Response != nil {
			ctx.Variables["resp"] = cty.MapVal(map[string]cty.Value{
				"headers": newCtyHeadersMap(req.Response.Header),
				"cookies": newCtyCookiesMap(req.Response.Cookies()),
				//"params":  newCtyParametersMap(mux.Vars(request)),
			})
		}
	}

	if beresp != nil {
		ctx.Variables["bereq"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(beresp.Request.Header),
		})
		ctx.Variables["beresp"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(beresp.Header),
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
			ctyMap[strings.ToLower(k)] = cty.StringVal(v[0]) // TODO: ListVal??
		}
	}
	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func newCtyCookiesMap(cookies []*http.Cookie) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, cookie := range cookies {
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

func decodeEnvironmentRefs(src []byte) []string {
	tokens, diags := hclsyntax.LexConfig(src, "tmp.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		panic(diags)
	}
	needle := []byte("env")
	var keys []string
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenIdent &&
			bytes.Equal(token.Bytes, needle) &&
			i+2 < len(tokens) {
			value := string(tokens[i+2].Bytes)
			if sort.SearchStrings(keys, value) == len(keys) {
				keys = append(keys, value)
			}
		}
	}
	return keys
}