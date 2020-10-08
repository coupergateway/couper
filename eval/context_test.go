package eval_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/seetie"
)

func TestNewHTTPContext(t *testing.T) {
	newBeresp := func(br *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "OK",
			Request:    br,
			Proto:      br.Proto,
			ProtoMajor: br.ProtoMajor,
			ProtoMinor: br.ProtoMinor,
		}
	}

	type expMap map[string]string
	type header map[string]string

	baseCtx := eval.NewENVContext(nil)

	tests := []struct {
		name      string
		reqMethod string
		reqHeader header
		body      io.Reader
		query     string
		baseCtx   *hcl.EvalContext
		hcl       string
		want      expMap
	}{
		{"Variables / POST", http.MethodPost, header{"Content-Type": "application/x-www-form-urlencoded"}, bytes.NewBufferString(`user=hans`), "", baseCtx, `
					post = req.post.user[0]
					method = req.method
		`, expMap{"post": "hans", "method": http.MethodPost}},
		{"Variables / Query", http.MethodGet, header{"User-Agent": "test/v1"}, nil, "?name=peter", baseCtx, `
					query = req.query.name[0]
					method = req.method
					ua = req.headers.user-agent
				`, expMap{"query": "peter", "method": http.MethodGet, "ua": "test/v1"}},
		{"Variables / Headers", http.MethodGet, header{"User-Agent": "test/v1"}, nil, "", baseCtx, `
					ua = req.headers.user-agent
					method = req.method
				`, expMap{"ua": "test/v1", "method": http.MethodGet}},
		{"Variables / PATCH /w json body", http.MethodPatch, header{"Content-Type": "application/json;charset=utf-8"}, bytes.NewBufferString(`{
			"obj_slice": [
				{"no_title": 123},
				{"title": "success"}
			]
}`), "", baseCtx, `
			method = req.method
			title = req.json_body.obj_slice[1].title
		`, expMap{"title": "success", "method": http.MethodPatch}},
		{"Variables / PATCH /w json body /wo CT header", http.MethodPatch, nil, bytes.NewBufferString(`{"slice": [1, 2, 3]}`), "", baseCtx, `
			method = req.method
			title = req.json_body.obj_slice
		`, expMap{"title": "", "method": http.MethodPatch}},
	}

	log, _ := test.NewNullLogger()

	// ensure proxy bufferOption for setBodyFunc
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			eval.ClientRequest: hcltest.MockExprTraversalSrc(eval.ClientRequest + "." + eval.JsonBody),
		}),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.reqMethod, "https://couper.io/"+tt.query, tt.body)
			*req = *req.Clone(context.WithValue(req.Context(), request.Endpoint, "couper-proxy"))

			for k, v := range tt.reqHeader {
				req.Header.Set(k, v)
			}

			bereq := req.Clone(context.Background())
			beresp := newBeresp(bereq)

			// since the proxy prepares the getBody rewind:
			proxy, err := handler.NewProxy(&handler.ProxyOptions{
				Context:          []hcl.Body{hclBody},
				RequestBodyLimit: 256,
				Origin:           req.URL.String(),
				CORS:             &handler.CORSOptions{},
			}, log.WithContext(nil), nil)
			if err != nil {
				t.Fatal(err)
			}
			proxyType := proxy.(*handler.Proxy)
			if err = proxyType.SetGetBody(req); err != nil {
				t.Fatal(err)
			}

			ctx := eval.NewHTTPContext(tt.baseCtx, eval.BufferRequest, req, bereq, beresp)
			ctx.Functions = nil // we are not interested in a functions test

			var resultMap map[string]cty.Value
			err = hclsimple.Decode("test.hcl", []byte(tt.hcl), ctx, &resultMap)
			// Expect same behaviour as in proxy impl and downgrade missing map elements to warnings.
			if err != nil && seetie.SetSeverityLevel(err.(hcl.Diagnostics)).HasErrors() {
				t.Fatal(err)
			}

			for k, v := range tt.want {
				cv, ok := resultMap[k]
				if !ok {
					t.Errorf("Expected value for %q, got nothing", k)
				}

				cvt := cv.Type()

				if cvt != cty.String && v == "" { // expected nothing, go ahead
					continue
				}

				if cvt != cty.String {
					t.Fatalf("Expected string value for %q, got %v", k, cvt)
				}

				if v != cv.AsString() {
					t.Errorf("%q want: %v, got: %#v", k, v, cv)
				}
			}
		})
	}
}
