package eval

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

type contextKey uint8

const ContextType contextKey = iota

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
	memorize     map[string]interface{}
	profiles     []*config.JWTSigningProfile
	saml         []*config.SAML
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
		},
		inner: context.TODO(), // usually replaced with request context
	}
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
		inner:        c.inner,
		memorize:     make(map[string]interface{}),
		profiles:     c.profiles[:],
		saml:         c.saml[:],
	}

	if rc := req.Context(); rc != nil {
		ctx.inner = context.WithValue(rc, ContextType, ctx)
	}

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

	body, jsonBody := parseReqBody(req)
	ctx.eval.Variables[ClientRequest] = cty.ObjectVal(ctxMap.Merge(ContextMap{
		FormBody:  seetie.ValuesMapToValue(parseForm(req).PostForm),
		ID:        cty.StringVal(id),
		Body:      body,
		JsonBody:  jsonBody,
		Method:    cty.StringVal(req.Method),
		Path:      cty.StringVal(req.URL.Path),
		PathParam: seetie.MapToValue(pathParams),
		Query:     seetie.ValuesMapToValue(req.URL.Query()),
		URL:       cty.StringVal(newRawURL(req.URL).String()),
	}.Merge(newVariable(ctx.inner, req.Cookies(), req.Header))))

	ctx.updateFunctions()

	return ctx
}

func (c *Context) WithBeresps(beresps ...*http.Response) *Context {
	ctx := &Context{
		bufferOption: c.bufferOption,
		eval:         cloneContext(c.eval),
		inner:        c.inner,
		memorize:     c.memorize,
		profiles:     c.profiles[:],
		saml:         c.saml[:],
	}
	ctx.inner = context.WithValue(c.inner, ContextType, ctx)

	resps := make(ContextMap)
	bereqs := make(ContextMap)
	for _, beresp := range beresps {
		if beresp == nil {
			continue
		}

		bereq := beresp.Request
		name := BackendDefault // TODO: name related error handling? override previous one for now
		if n, ok := bereq.Context().Value(request.RoundTripName).(string); ok {
			name = n
		}
		bereqs[name] = cty.ObjectVal(ContextMap{
			FormBody: seetie.ValuesMapToValue(parseForm(bereq).PostForm),
			Method:   cty.StringVal(bereq.Method),
			Path:     cty.StringVal(bereq.URL.Path),
			Query:    seetie.ValuesMapToValue(bereq.URL.Query()),
			URL:      cty.StringVal(newRawURL(bereq.URL).String()),
		}.Merge(newVariable(ctx.inner, bereq.Cookies(), bereq.Header)))

		var body, jsonBody cty.Value
		if (ctx.bufferOption & BufferResponse) == BufferResponse {
			body, jsonBody = parseRespBody(beresp)
		}
		resps[name] = cty.ObjectVal(ContextMap{
			HttpStatus: cty.StringVal(strconv.Itoa(beresp.StatusCode)),
			JsonBody:   jsonBody,
			Body:       body,
		}.Merge(newVariable(ctx.inner, beresp.Cookies(), beresp.Header)))
	}

	ctx.eval.Variables[BackendRequests] = cty.ObjectVal(bereqs)
	ctx.eval.Variables[BackendResponses] = cty.ObjectVal(resps)

	ctx.updateFunctions()

	return ctx
}

// WithJWTProfiles initially setup the lib.FnJWTSign function.
func (c *Context) WithJWTProfiles(profiles []*config.JWTSigningProfile) *Context {
	c.profiles = profiles
	if c.profiles == nil {
		c.profiles = make([]*config.JWTSigningProfile, 0)
	}
	c.updateFunctions()
	return c
}

// WithSAML initially setup the lib.FnSamlSsoUrl function.
func (c *Context) WithSAML(s []*config.SAML) *Context {
	c.saml = s
	if c.saml == nil {
		c.saml = make([]*config.SAML, 0)
	}
	c.updateFunctions()
	return c
}

func (c *Context) HCLContext() *hcl.EvalContext {
	return c.eval
}

// updateFunctions recreates the listed functions with latest evaluation context.
func (c *Context) updateFunctions() {
	if len(c.profiles) > 0 {
		jwtfn := lib.NewJwtSignFunction(c.profiles, c.eval)
		c.eval.Functions[lib.FnJWTSign] = jwtfn
	}
	if len(c.saml) > 0 {
		samlfn := lib.NewSamlSsoUrlFunction(c.saml)
		c.eval.Functions[lib.FnSamlSsoUrl] = samlfn
	}

	c.eval.Functions[lib.FnOAuthCodeVerifier] = lib.NewOAuthCodeVerifierFunction(c.getCodeVerifier)
	c.eval.Functions[lib.FnOAuthCodeChallenge] = lib.NewOAuthCodeChallengeFunction(c.getCodeVerifier)
}

func (c *Context) getCodeVerifier() (*pkce.CodeVerifier, error) {
	cv, ok := c.memorize[lib.CodeVerifier]
	var err error
	if !ok {
		cv, err = pkce.CreateCodeVerifier()
		if err != nil {
			return nil, err
		}

		c.memorize[lib.CodeVerifier] = cv
	}

	codeVerifier, _ := cv.(*pkce.CodeVerifier)

	return codeVerifier, nil
}

const defaultMaxMemory = 32 << 20 // 32 MB

// parseForm populates the request PostForm field.
// As Proxy we should not consume the request body.
// Rewind body via GetBody method.
func parseForm(r *http.Request) *http.Request {
	if r.GetBody == nil || r.Form != nil {
		return r
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		_ = r.ParseMultipartForm(defaultMaxMemory)
		r.Body, _ = r.GetBody() // reset
	}
	return r
}

func isJSONMediaType(contentType string) bool {
	m, _, _ := mime.ParseMediaType(contentType)
	mParts := strings.Split(m, "/")
	return len(mParts) == 2 && mParts[0] == "application" && (mParts[1] == "json" || strings.HasSuffix(mParts[1], "+json"))
}

func parseReqBody(req *http.Request) (cty.Value, cty.Value) {
	jsonBody := cty.EmptyObjectVal
	if req == nil || req.GetBody == nil {
		return cty.NilVal, jsonBody
	}

	body, _ := req.GetBody()
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return cty.NilVal, jsonBody
	}

	if isJSONMediaType(req.Header.Get("Content-Type")) {
		jsonBody = parseJSONBytes(b)
	}
	return cty.StringVal(string(b)), jsonBody
}

func parseRespBody(beresp *http.Response) (cty.Value, cty.Value) {
	jsonBody := cty.EmptyObjectVal

	if beresp == nil || beresp.Body == nil {
		return cty.NilVal, jsonBody
	}

	b, err := ioutil.ReadAll(beresp.Body)
	if err != nil {
		return cty.NilVal, jsonBody
	}

	beresp.Body = io.NopCloser(bytes.NewBuffer(b)) // reset

	if isJSONMediaType(beresp.Header.Get("Content-Type")) {
		jsonBody = parseJSONBytes(b)
	}
	return cty.StringVal(string(b)), jsonBody
}

func parseJSONBytes(b []byte) cty.Value {
	impliedType, err := ctyjson.ImpliedType(b)
	if err != nil {
		return cty.EmptyObjectVal
	}

	val, err := ctyjson.Unmarshal(b, impliedType)
	if err != nil {
		return cty.EmptyObjectVal
	}
	return val
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
	acData, _ := ctx.Value(request.AccessControls).(map[string]interface{})
	ctxAcMap := make(map[string]cty.Value)
	for name, data := range acData {
		dataMap, ok := data.(map[string]interface{})
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
		"merge":         lib.MergeFunc,
		"to_lower":      stdlib.LowerFunc,
		"to_upper":      stdlib.UpperFunc,
		"unixtime":      lib.UnixtimeFunc,
		"url_encode":    lib.UrlEncodeFunc,
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
