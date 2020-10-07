package eval

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/request"
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

	baseCtx := NewENVContext(nil)

	tests := []struct {
		name      string
		reqMethod string
		body      io.Reader
		query     string
		baseCtx   *hcl.EvalContext
		hcl       string
		want      expMap
	}{
		{"Variables / POST", http.MethodPost, strings.NewReader(`user=hans`), "", baseCtx, `
			post = req.post.user[0]
			method = req.method
`, expMap{"post": "hans", "method": http.MethodPost}},
		{"Variables / Query", http.MethodGet, nil, "?name=peter", baseCtx, `
			query = req.query.name[0]
			method = req.method
			ua = req.headers.user-agent
		`, expMap{"query": "peter", "method": http.MethodGet, "ua": "test/v1"}},
		{"Variables / Headers", http.MethodGet, nil, "", baseCtx, `
			ua = req.headers.user-agent
			method = req.method
		`, expMap{"ua": "test/v1", "method": http.MethodGet}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.reqMethod, "https://couper.io/"+tt.query, tt.body)
			*req = *req.Clone(context.WithValue(req.Context(), request.Endpoint, "couper-proxy"))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			req.Header.Set("User-Agent", "test/v1")

			bereq := req.Clone(context.Background())
			beresp := newBeresp(bereq)

			ctx := NewHTTPContext(tt.baseCtx, BufferRequest, req, bereq, beresp)
			ctx.Functions = nil // we are not interested in a functions test

			var resultMap map[string]cty.Value
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), ctx, &resultMap)
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range tt.want {
				cv, ok := resultMap[k]
				if !ok {
					t.Errorf("Expected value for %q, got nothing", k)
				}

				cvt := cv.Type()
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
