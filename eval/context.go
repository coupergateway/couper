package eval

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

type Roundtrip interface {
	Context() context.Context
	Cookies() []*http.Cookie
}

type ContextMap map[string]cty.Value

func (m ContextMap) Merge(other ContextMap) ContextMap {
	for k, v := range other {
		m[k] = v
	}
	return m
}

func NewENVContext(src []byte) *hcl.EvalContext {
	envKeys := decodeEnvironmentRefs(src)
	variables := make(map[string]cty.Value)
	variables["env"] = newCtyEnvMap(envKeys)

	return &hcl.EvalContext{
		Variables: variables,
		Functions: newFunctionsMap(),
	}
}

func NewHTTPContext(baseCtx *hcl.EvalContext, req, bereq *http.Request, beresp *http.Response) *hcl.EvalContext {
	if req == nil {
		return baseCtx
	}
	evalCtx := cloneContext(baseCtx)
	httpCtx := req.Context()

	reqCtxMap := ContextMap{}
	if endpoint, ok := httpCtx.Value(request.Endpoint).(string); ok {
		reqCtxMap["endpoint"] = cty.StringVal(endpoint)
	}

	var id string
	if uid, ok := httpCtx.Value(request.UID).(string); ok {
		id = uid
	}

	evalCtx.Variables["req"] = cty.ObjectVal(reqCtxMap.Merge(ContextMap{
		"id":     cty.StringVal(id),
		"method": cty.StringVal(req.Method),
		"path":   cty.StringVal(req.URL.Path),
		"url":    cty.StringVal(newRawURL(req.URL).String()),
		"query":  seetie.ValuesMapToValue(req.URL.Query()),
		"post":   seetie.ValuesMapToValue(req.PostForm),
	}.Merge(newVariable(httpCtx, req.Cookies(), req.Header))))

	if beresp != nil {
		evalCtx.Variables["bereq"] = cty.ObjectVal(ContextMap{
			"method": cty.StringVal(bereq.Method),
			"path":   cty.StringVal(bereq.URL.Path),
			"url":    cty.StringVal(newRawURL(bereq.URL).String()),
			"query":  seetie.ValuesMapToValue(bereq.URL.Query()),
			"post":   seetie.ValuesMapToValue(bereq.PostForm),
		}.
			Merge(newVariable(httpCtx, bereq.Cookies(), bereq.Header)))
		evalCtx.Variables["beresp"] = cty.ObjectVal(ContextMap{
			"status": cty.StringVal(strconv.Itoa(beresp.StatusCode)),
		}.Merge(newVariable(httpCtx, beresp.Cookies(), beresp.Header)))
	}

	return evalCtx
}

func newRawURL(u *url.URL) *url.URL {
	rawURL := *u
	rawURL.RawQuery = ""
	rawURL.Fragment = ""
	return &rawURL
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

func newVariable(ctx context.Context, cookies []*http.Cookie, headers http.Header) ContextMap {
	jwtClaims, _ := ctx.Value(ac.ContextAccessControlKey).(map[string]interface{})
	ctxAcMap := make(map[string]cty.Value)
	for name, data := range jwtClaims {
		dataMap, ok := data.(ac.Claims)
		if !ok {
			continue
		}
		ctxAcMap[name] = seetie.MapToValue(dataMap)
	}
	var ctxAcMapValue cty.Value
	if len(ctxAcMap) > 0 {
		ctxAcMapValue = cty.MapVal(ctxAcMap)
	} else {
		ctxAcMapValue = cty.MapValEmpty(cty.String)
	}

	return map[string]cty.Value{
		"ctx":     ctxAcMapValue,
		"cookies": seetie.CookiesToMapValue(cookies),
		"headers": seetie.HeaderToMapValue(headers),
	}
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

// Functions
func newFunctionsMap() map[string]function.Function {
	return map[string]function.Function{
		"base64_decode": lib.Base64DecodeFunc,
		"base64_encode": lib.Base64EncodeFunc,
		"to_upper":      stdlib.UpperFunc,
		"to_lower":      stdlib.LowerFunc,
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
