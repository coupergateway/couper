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

func NewHTTPContext(baseCtx *hcl.EvalContext, bufOpt BufferOption, req, bereq *http.Request, beresp *http.Response) *hcl.EvalContext {
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

	if (bufOpt & BufferRequest) == BufferRequest {
		switch req.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			setGetBody(req)
		}
	}

	evalCtx.Variables["req"] = cty.ObjectVal(reqCtxMap.Merge(ContextMap{
		"id":        cty.StringVal(id),
		"method":    cty.StringVal(req.Method),
		"path":      cty.StringVal(req.URL.Path),
		"url":       cty.StringVal(newRawURL(req.URL).String()),
		"query":     seetie.ValuesMapToValue(req.URL.Query()),
		"post":      seetie.ValuesMapToValue(parseForm(req).PostForm),
		"json_body": seetie.MapToValue(parseReqJSON(req)),
	}.Merge(newVariable(httpCtx, req.Cookies(), req.Header))))

	if beresp != nil {
		evalCtx.Variables["bereq"] = cty.ObjectVal(ContextMap{
			"method": cty.StringVal(bereq.Method),
			"path":   cty.StringVal(bereq.URL.Path),
			"url":    cty.StringVal(newRawURL(bereq.URL).String()),
			"query":  seetie.ValuesMapToValue(bereq.URL.Query()),
			"post":   seetie.ValuesMapToValue(parseForm(bereq).PostForm),
		}.Merge(newVariable(httpCtx, bereq.Cookies(), bereq.Header)))

		var jsonBody map[string]interface{}
		if (bufOpt & BufferResponse) == BufferResponse {
			jsonBody = parseRespJSON(beresp)
		}
		evalCtx.Variables["beresp"] = cty.ObjectVal(ContextMap{
			"status":    cty.StringVal(strconv.Itoa(beresp.StatusCode)),
			"json_body": seetie.MapToValue(jsonBody),
		}.Merge(newVariable(httpCtx, beresp.Cookies(), beresp.Header)))
	}

	return evalCtx
}

const defaultMaxMemory = 32 << 20 // 32 MB

type readCloser struct {
	io.Reader
	closer io.Closer
}

func newReadCloser(r io.Reader, c io.Closer) *readCloser {
	return &readCloser{Reader: r, closer: c}
}

func (rc readCloser) Close() error {
	return rc.closer.Close()
}

func setGetBody(r *http.Request) *http.Request {
	if r.Body != nil && r.GetBody == nil {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		r.GetBody = func() (io.ReadCloser, error) {
			return newReadCloser(bytes.NewBuffer(bodyBytes), r.Body), nil
		}
		// reset
		r.Body, _ = r.GetBody()
	}
	return r
}

// parseForm populates the request PostForm field.
// As Proxy we should not consume the request body.
// Rewind body via GetBody method.
func parseForm(r *http.Request) *http.Request {
	if r.GetBody == nil {
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
	beresp.Body = newReadCloser(bytes.NewBuffer(buf.Bytes()), beresp.Body)
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
