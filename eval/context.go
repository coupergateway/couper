package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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

const ContextType = "evalContextType"

var _ context.Context = &Context{}

type ContextMap map[string]cty.Value

func (m ContextMap) Merge(other ContextMap) ContextMap {
	for k, v := range other {
		m[k] = v
	}
	return m
}

type Context struct {
	bufferOption BufferOption
	eval         *hcl.EvalContext
	inner        context.Context
}

func NewContext(src []byte) *Context {
	envKeys := decodeEnvironmentRefs(src)
	variables := make(map[string]cty.Value)
	variables[Environment] = newCtyEnvMap(envKeys)

	return &Context{
		bufferOption: BufferRequest | BufferResponse, // TODO: eval per endpoint body route
		eval: &hcl.EvalContext{
			Variables: variables,
			Functions: newFunctionsMap(),
		}}
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.inner.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.inner.Done()
}

func (c *Context) Err() error {
	return c.inner.Err()
}

func (c *Context) Value(key interface{}) interface{} {
	if key == ContextType {
		return c
	}
	return c.inner.Value(key)
}

func (c *Context) WithClientRequest(req *http.Request) *Context {
	ctx := &Context{
		bufferOption: c.bufferOption,
		eval:         cloneContext(c.eval),
	}
	ctx.inner = context.WithValue(req.Context(), ContextType, ctx)

	ctxMap := ContextMap{}
	if endpoint, ok := ctx.inner.Value(request.Endpoint).(string); ok {
		ctxMap[Endpoint] = cty.StringVal(endpoint)
	}

	var id string
	if uid, ok := ctx.inner.Value(request.UID).(string); ok {
		id = uid
	}

	var pathParams request.PathParameter
	if params, ok := ctx.inner.Value(request.PathParams).(request.PathParameter); ok {
		pathParams = params
	}

	ctx.eval.Variables[ClientRequest] = cty.ObjectVal(ctxMap.Merge(ContextMap{
		ID:        cty.StringVal(id),
		JsonBody:  seetie.MapToValue(parseReqJSON(req)),
		Method:    cty.StringVal(req.Method),
		Path:      cty.StringVal(req.URL.Path),
		PathParam: seetie.MapToValue(pathParams),
		Post:      seetie.ValuesMapToValue(parseForm(req).PostForm),
		Query:     seetie.ValuesMapToValue(req.URL.Query()),
		URL:       cty.StringVal(newRawURL(req.URL).String()),
	}.Merge(newVariable(ctx.inner, req.Cookies(), req.Header))))

	return ctx
}

func (c *Context) WithBeresps(beresps ...*http.Response) *Context {
	ctx := &Context{
		bufferOption: c.bufferOption,
		eval:         cloneContext(c.eval),
	}
	ctx.inner = context.WithValue(c.inner, ContextType, ctx)

	resps := make(ContextMap, 0)
	bereqs := make(ContextMap, 0)
	for _, beresp := range beresps {
		bereq := beresp.Request
		name := BackendDefault // TODO: name related error handling? override previous one for now
		if n, ok := bereq.Context().Value(request.RoundTripName).(string); ok {
			name = n
		}
		bereqs[name] = cty.ObjectVal(ContextMap{
			Method: cty.StringVal(bereq.Method),
			Path:   cty.StringVal(bereq.URL.Path),
			Post:   seetie.ValuesMapToValue(parseForm(bereq).PostForm),
			Query:  seetie.ValuesMapToValue(bereq.URL.Query()),
			URL:    cty.StringVal(newRawURL(bereq.URL).String()),
		}.Merge(newVariable(ctx.inner, bereq.Cookies(), bereq.Header)))

		var jsonBody map[string]interface{}
		if (ctx.bufferOption & BufferResponse) == BufferResponse {
			jsonBody = parseRespJSON(beresp)
		}
		resps[name] = cty.ObjectVal(ContextMap{
			HttpStatus: cty.StringVal(strconv.Itoa(beresp.StatusCode)),
			JsonBody:   seetie.MapToValue(jsonBody),
		}.Merge(newVariable(ctx.inner, beresp.Cookies(), beresp.Header)))
	}

	if val, ok := bereqs[BackendDefault]; ok {
		ctx.eval.Variables[BackendRequest] = val
	}
	if val, ok := resps[BackendDefault]; ok {
		ctx.eval.Variables[BackendResponse] = val
	}
	ctx.eval.Variables[BackendRequests] = cty.ObjectVal(bereqs)
	ctx.eval.Variables[BackendResponses] = cty.ObjectVal(resps)

	return ctx
}

func (c Context) HCLContext() *hcl.EvalContext {
	return c.eval
}

const defaultMaxMemory = 32 << 20 // 32 MB

// parseForm populates the request PostForm field.
// As Proxy we should not consume the request body.
// Rewind body via GetBody method.
func parseForm(r *http.Request) *http.Request {
	if r.GetBody == nil {
		return r
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		r.Body, _ = r.GetBody() // rewind
		_ = r.ParseMultipartForm(defaultMaxMemory)
		r.Body, _ = r.GetBody() // reset
	}
	return r
}

func isJSONMediaType(contentType string) bool {
	m, _, _ := mime.ParseMediaType(contentType)
	return m == "application/json"
}

func parseJSON(r io.Reader) map[string]interface{} {
	if r == nil {
		return nil
	}
	var result map[string]interface{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(b, &result)
	return result
}

func parseReqJSON(req *http.Request) map[string]interface{} {
	if req.GetBody == nil {
		return nil
	}

	if !isJSONMediaType(req.Header.Get("Content-Type")) {
		return nil
	}

	req.Body, _ = req.GetBody() // rewind
	result := parseJSON(req.Body)
	req.Body, _ = req.GetBody() // reset
	return result
}

func parseRespJSON(beresp *http.Response) map[string]interface{} {
	if !isJSONMediaType(beresp.Header.Get("Content-Type")) {
		return nil
	}

	buf := &bytes.Buffer{}
	io.Copy(buf, beresp.Body) // TODO: err handling
	// reset
	beresp.Body = NewReadCloser(bytes.NewBuffer(buf.Bytes()), beresp.Body)
	return parseJSON(buf)
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
		CTX:     ctxAcMapValue,
		Cookies: seetie.CookiesToMapValue(cookies),
		Headers: seetie.HeaderToMapValue(headers),
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
		"coalesce":      stdlib.CoalesceFunc,
		"json_decode":   stdlib.JSONDecodeFunc,
		"json_encode":   stdlib.JSONEncodeFunc,
		"to_lower":      stdlib.LowerFunc,
		"to_upper":      stdlib.UpperFunc,
		"unixtime":      lib.UnixtimeFunc,
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
		if token.Type == hclsyntax.TokenDot && i > 0 &&
			bytes.Equal(tokens[i-1].Bytes, needle) &&
			i+1 < len(tokens) {
			value := string(tokens[i+1].Bytes)
			if !hasValue(keys, value) {
				keys = append(keys, value)
			}
		}
	}
	return keys
}

func hasValue(list []string, needle string) bool {
	for _, s := range list {
		if s == needle {
			return true
		}
	}
	return false
}
