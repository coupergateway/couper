package eval

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/oauth2/oidc"
	"github.com/avenga/couper/utils"
)

var _ context.Context = &Context{}

type ContextMap map[string]cty.Value

func (m ContextMap) Merge(other ContextMap) ContextMap {
	for k, v := range other {
		m[k] = v
	}
	return m
}

type Context struct {
	bufferOption      BufferOption
	eval              *hcl.EvalContext
	inner             context.Context
	memorize          map[string]interface{}
	oauth2            []config.OAuth2Authorization
	jwtSigningConfigs map[string]*lib.JWTSigningConfig
	saml              []*config.SAML
	src               []byte
}

func NewContext(src []byte, defaults *config.Defaults) *Context {
	defaultEnvVariables := make(config.DefaultEnvVars)
	if defaults != nil {
		defaultEnvVariables = defaults.EnvironmentVariables
	}

	variables := make(map[string]cty.Value)
	variables[Environment] = newCtyEnvMap(defaultEnvVariables)
	variables[Couper] = newCtyCouperVariablesMap()

	return &Context{
		bufferOption: BufferRequest | BufferResponse, // TODO: eval per endpoint body route
		eval: &hcl.EvalContext{
			Variables: variables,
			Functions: newFunctionsMap(),
		},
		inner: context.TODO(), // usually replaced with request context
		src:   src,
	}
}

// ContextFromRequest extracts the eval.Context implementation value from given request and
// returns a noop one as fallback.
func ContextFromRequest(req *http.Request) *Context {
	if evalCtx, ok := req.Context().Value(request.ContextType).(*Context); ok {
		return evalCtx
	}
	return NewContext(nil, nil)
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
	if key == request.ContextType {
		return c
	}
	return c.inner.Value(key)
}

func (c *Context) WithClientRequest(req *http.Request) *Context {
	ctx := &Context{
		bufferOption:      c.bufferOption,
		eval:              c.cloneEvalContext(),
		inner:             c.inner,
		memorize:          make(map[string]interface{}),
		oauth2:            c.oauth2[:],
		jwtSigningConfigs: c.jwtSigningConfigs,
		saml:              c.saml[:],
	}

	if rc := req.Context(); rc != nil {
		ctx.inner = context.WithValue(rc, request.ContextType, ctx)
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

	p := req.URL.Port()
	if p == "" {
		if req.URL.Scheme == "https" {
			p = "443"
		} else {
			p = "80"
		}
	}
	port, _ := strconv.ParseInt(p, 10, 64)
	body, jsonBody := parseReqBody(req)

	origin := NewRawOrigin(req.URL)
	ctx.eval.Variables[ClientRequest] = cty.ObjectVal(ctxMap.Merge(ContextMap{
		ID:        cty.StringVal(id),
		Method:    cty.StringVal(req.Method),
		PathParam: seetie.MapToValue(pathParams),
		URL:       cty.StringVal(req.URL.String()),
		Origin:    cty.StringVal(origin.String()),
		Protocol:  cty.StringVal(req.URL.Scheme),
		Host:      cty.StringVal(req.URL.Hostname()),
		Port:      cty.NumberIntVal(port),
		Path:      cty.StringVal(req.URL.Path),
		Query:     seetie.ValuesMapToValue(req.URL.Query()),
		Body:      body,
		JsonBody:  jsonBody,
		FormBody:  seetie.ValuesMapToValue(parseForm(req).PostForm),
	}.Merge(newVariable(ctx.inner, req.Cookies(), req.Header))))

	ctx.updateRequestRelatedFunctions(origin)
	ctx.updateFunctions()

	return ctx
}

func (c *Context) WithBeresps(beresps ...*http.Response) *Context {
	ctx := &Context{
		bufferOption:      c.bufferOption,
		eval:              c.cloneEvalContext(),
		inner:             c.inner,
		memorize:          c.memorize,
		oauth2:            c.oauth2[:],
		jwtSigningConfigs: c.jwtSigningConfigs,
		saml:              c.saml[:],
	}
	ctx.inner = context.WithValue(c.inner, request.ContextType, ctx)

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

		p := bereq.URL.Port()
		if p == "" {
			if bereq.URL.Scheme == "https" {
				p = "443"
			} else {
				p = "80"
			}
		}
		port, _ := strconv.ParseInt(p, 10, 64)

		body, jsonBody := parseReqBody(bereq)
		bereqs[name] = cty.ObjectVal(ContextMap{
			Method:   cty.StringVal(bereq.Method),
			URL:      cty.StringVal(bereq.URL.String()),
			Origin:   cty.StringVal(NewRawOrigin(bereq.URL).String()),
			Protocol: cty.StringVal(bereq.URL.Scheme),
			Host:     cty.StringVal(bereq.URL.Hostname()),
			Port:     cty.NumberIntVal(port),
			Path:     cty.StringVal(bereq.URL.Path),
			Query:    seetie.ValuesMapToValue(bereq.URL.Query()),
			Body:     body,
			JsonBody: jsonBody,
			FormBody: seetie.ValuesMapToValue(parseForm(bereq).PostForm),
		}.Merge(newVariable(ctx.inner, bereq.Cookies(), bereq.Header)))

		var respBody, respJsonBody cty.Value
		if !IsUpgradeResponse(bereq, beresp) {
			if (ctx.bufferOption & BufferResponse) == BufferResponse {
				respBody, respJsonBody = parseRespBody(beresp)
			}
		}
		resps[name] = cty.ObjectVal(ContextMap{
			HttpStatus: cty.NumberIntVal(int64(beresp.StatusCode)),
			JsonBody:   respJsonBody,
			Body:       respBody,
		}.Merge(newVariable(ctx.inner, beresp.Cookies(), beresp.Header)))
	}

	ctx.eval.Variables[BackendRequests] = cty.ObjectVal(bereqs)
	ctx.eval.Variables[BackendResponses] = cty.ObjectVal(resps)

	clientOrigin, _ := seetie.ValueToMap(ctx.eval.Variables[ClientRequest])[Origin].(string)
	originUrl, _ := url.Parse(clientOrigin)
	ctx.updateRequestRelatedFunctions(originUrl)
	ctx.updateFunctions()

	return ctx
}

// WithJWTSigningConfigs initially sets up the lib.FnJWTSign function.
func (c *Context) WithJWTSigningConfigs(configs map[string]*lib.JWTSigningConfig) *Context {
	c.jwtSigningConfigs = configs
	if c.jwtSigningConfigs == nil {
		c.jwtSigningConfigs = make(map[string]*lib.JWTSigningConfig, 0)
	}
	c.updateFunctions()
	return c
}

// WithOAuth2AC adds the OAuth2AC config structs.
func (c *Context) WithOAuth2AC(os []*config.OAuth2AC) *Context {
	if c.oauth2 == nil {
		c.oauth2 = make([]config.OAuth2Authorization, 0)
	}
	for _, o := range os {
		c.oauth2 = append(c.oauth2, o)
	}
	return c
}

// WithOidcConfig adds the OidcConfig config structs.
func (c *Context) WithOidcConfig(confs oidc.Configs) *Context {
	for _, oidcConf := range confs {
		c.oauth2 = append(c.oauth2, oidcConf)
	}
	if c.oauth2 == nil {
		c.oauth2 = make([]config.OAuth2Authorization, 0)
	}
	return c
}

// WithSAML initially set up the saml configuration.
func (c *Context) WithSAML(s []*config.SAML) *Context {
	c.saml = s
	if c.saml == nil {
		c.saml = make([]*config.SAML, 0)
	}
	return c
}

func (c *Context) HCLContext() *hcl.EvalContext {
	return c.eval
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

// updateFunctions recreates the listed functions with the current evaluation context.
func (c *Context) updateFunctions() {
	jwtfn := lib.NewJwtSignFunction(c.eval, c.jwtSigningConfigs, Value)
	c.eval.Functions[lib.FnJWTSign] = jwtfn
}

// updateRequestRelatedFunctions re-creates the listed functions for the client request context.
func (c *Context) updateRequestRelatedFunctions(origin *url.URL) {
	if c.oauth2 != nil {
		oauth2fn := lib.NewOAuthAuthorizationUrlFunction(c.eval, c.oauth2, c.getCodeVerifier, origin)
		c.eval.Functions[lib.FnOAuthAuthorizationUrl] = oauth2fn
	}
	c.eval.Functions[lib.FnOAuthVerifier] = lib.NewOAuthCodeVerifierFunction(c.getCodeVerifier)
	c.eval.Functions[lib.InternalFnOAuthHashedVerifier] = lib.NewOAuthCodeChallengeFunction(c.getCodeVerifier)

	if c.saml != nil {
		samlfn := lib.NewSamlSsoUrlFunction(c.saml, origin)
		c.eval.Functions[lib.FnSamlSsoUrl] = samlfn
	}
}

func (c *Context) cloneEvalContext() *hcl.EvalContext {
	ctx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: make(map[string]function.Function),
	}

	for key, val := range c.eval.Variables {
		ctx.Variables[key] = val
	}

	for key, val := range c.eval.Functions {
		ctx.Functions[key] = val
	}

	return ctx
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
	b, err := io.ReadAll(body)
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

	b, err := io.ReadAll(beresp.Body)
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

func NewRawOrigin(u *url.URL) *url.URL {
	rawOrigin := *u
	rawOrigin.Path = ""
	rawOrigin.RawQuery = ""
	rawOrigin.Fragment = ""
	return &rawOrigin
}

func newVariable(ctx context.Context, cookies []*http.Cookie, headers http.Header) ContextMap {
	acData, _ := ctx.Value(request.AccessControls).(map[string]interface{})
	scopeData, _ := ctx.Value(request.Scopes).([]string)
	ctxAcMap := make(map[string]cty.Value)
	for name, data := range acData {
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			continue
		}
		ctxAcMap[name] = seetie.MapToValue(dataMap)
	}
	if len(scopeData) > 0 {
		ctxAcMap["scopes"] = seetie.GoToValue(scopeData)
	}
	var ctxAcMapValue cty.Value
	if len(ctxAcMap) > 0 {
		ctxAcMapValue = cty.ObjectVal(ctxAcMap)
	} else {
		ctxAcMapValue = cty.MapValEmpty(cty.String)
	}

	return map[string]cty.Value{
		CTX:     ctxAcMapValue,
		Cookies: seetie.CookiesToMapValue(cookies),
		Headers: seetie.HeaderToMapValue(headers),
	}
}

func newCtyEnvMap(defaultValues map[string]string) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, pair := range os.Environ() {
		key := strings.Split(pair, "=")[0]
		value := os.Getenv(key)
		if _, ok := ctyMap[key]; !ok {
			if val, set := defaultValues[key]; set && value == "" {
				ctyMap[key] = cty.StringVal(val)
				continue
			}
			if value != "" { // do not set empty string, fallback to nilVal per default
				ctyMap[key] = cty.StringVal(value)
			}
		}
	}
	return cty.MapVal(ctyMap)
}

func newCtyCouperVariablesMap() cty.Value {
	ctyMap := map[string]cty.Value{
		"version": cty.StringVal(utils.VersionName),
	}
	return cty.MapVal(ctyMap)
}

// Functions
func newFunctionsMap() map[string]function.Function {
	return map[string]function.Function{
		"base64_decode": lib.Base64DecodeFunc,
		"base64_encode": lib.Base64EncodeFunc,
		"coalesce":      lib.CoalesceFunc,
		"json_decode":   stdlib.JSONDecodeFunc,
		"json_encode":   stdlib.JSONEncodeFunc,
		"merge":         lib.MergeFunc,
		"to_lower":      stdlib.LowerFunc,
		"to_upper":      stdlib.UpperFunc,
		"unixtime":      lib.UnixtimeFunc,
		"url_encode":    lib.UrlEncodeFunc,
	}
}
