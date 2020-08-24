package eval

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	ac "go.avenga.cloud/couper/gateway/access_control"
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

func NewHTTPContext(baseCtx *hcl.EvalContext, req *http.Request, beresp *http.Response) *hcl.EvalContext {
	ctx := cloneContext(baseCtx)
	if req != nil {
		ctx.Variables["req"] = newVariable(req.Context(), req.Cookies(), req.Header)
	}

	if beresp != nil {
		bereq := beresp.Request
		ctx.Variables["bereq"] = newVariable(bereq.Context(), bereq.Cookies(), bereq.Header)
		ctx.Variables["beresp"] = newVariable(context.Background(), beresp.Cookies(), beresp.Header)
	}

	return ctx
}

func cloneContext(ctx *hcl.EvalContext) *hcl.EvalContext {
	c := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: make(map[string]function.Function),
	}

	for key, val := range ctx.Variables {
		c.Variables[key] = val
	}

	for key, val := range ctx.Functions {
		c.Functions[key] = val
	}
	return c
}

func newVariable(ctx context.Context, cookies []*http.Cookie, headers http.Header) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"ctx": cty.MapVal(map[string]cty.Value{
			newString(ctx.Value(ac.ContextAccessControlKey)): newCtyClaimsMap(ctx.Value(ac.ContextJWTClaimKey)),
		}),
		"cookies": newCtyCookiesMap(cookies),
		"headers": newCtyHeadersMap(headers),
	})
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

func newCtyClaimsMap(value interface{}) cty.Value {
	valueMap, ok := value.(map[string]interface{})
	if !ok {
		valueMap, ok = value.(ac.Claims)
	}
	if !ok {
		cty.MapValEmpty(cty.String)
	}
	ctyMap := make(map[string]cty.Value)
	for k, v := range valueMap {
		if isValidKey(k) {
			ctyMap[k] = cty.StringVal(newString(v))
		}
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func newString(s interface{}) string {
	switch s.(type) {
	case string:
		return s.(string)
	case int:
		return strconv.Itoa(s.(int))
	case float64:
		return fmt.Sprintf("%0.f", s)
	case bool:
		if !s.(bool) {
			return "false"
		}
		return "true"
	default:
		return ""
	}
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
