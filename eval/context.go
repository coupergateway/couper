package eval

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	ac "go.avenga.cloud/couper/gateway/access_control"
	"go.avenga.cloud/couper/gateway/eval/lib"
	"go.avenga.cloud/couper/gateway/internal/seetie"
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
	}
	return cty.ObjectVal(map[string]cty.Value{
		"ctx":     ctxAcMapValue,
		"cookies": seetie.CookiesToMapValue(cookies),
		"headers": seetie.HeaderToMapValue(headers),
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
