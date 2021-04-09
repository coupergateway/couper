package eval_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func init() {
	_ = os.Setenv("COUPER_TEST_VAR_SET", "")
	_ = os.Setenv("COUPER_TEST_VAR_VALUE_SET", "value")
}

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

	baseCtx := eval.NewContext(nil)

	tests := []struct {
		name      string
		reqMethod string
		reqHeader http.Header
		body      io.Reader
		query     string
		baseCtx   *eval.Context
		hcl       string
		want      http.Header
	}{
		{"Variables / POST", http.MethodPost, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, bytes.NewBufferString(`user=hans`), "", baseCtx, `
					post = request.form_body.user[0]
					method = request.method
		`, http.Header{"post": {"hans"}, "method": {http.MethodPost}}},
		{"Variables / Query", http.MethodGet, http.Header{"User-Agent": {"test/v1"}}, nil, "?name=peter", baseCtx, `
					query = request.query.name[0]
					method = request.method
					ua = request.headers.user-agent
				`, http.Header{"query": {"peter"}, "method": {http.MethodGet}, "ua": {"test/v1"}}},
		{"Variables / Headers", http.MethodGet, http.Header{"User-Agent": {"test/v1"}}, nil, "", baseCtx, `
					ua = request.headers.user-agent
					method = request.method
				`, http.Header{"ua": {"test/v1"}, "method": {http.MethodGet}}},
		{"Variables / PATCH /w json body", http.MethodPatch, http.Header{"Content-Type": {"application/json;charset=utf-8"}}, bytes.NewBufferString(`{
			"obj_slice": [
				{"no_title": 123},
				{"title": "success"}
			]}`), "", baseCtx, `
			method = request.method
			title = request.json_body.obj_slice[1].title
		`, http.Header{"title": {"success"}, "method": {http.MethodPatch}}},
		{"Variables / PATCH /w json body /wo CT header", http.MethodPatch, nil, bytes.NewBufferString(`{"slice": [1, 2, 3]}`), "", baseCtx, `
			method = request.method
			title = request.json_body.obj_slice
		`, http.Header{"method": {http.MethodPatch}}},
		{"Variables / GET /w json body", http.MethodGet, http.Header{"Content-Type": {"application/json"}}, bytes.NewBufferString(`{"slice": [1, 2, 3]}`), "", baseCtx, `
			method = request.method
			title = request.json_body.slice
		`, http.Header{"title": {"1", "2", "3"}, "method": {http.MethodGet}}},
		{"Variables / GET /w json body & missing attribute", http.MethodGet, http.Header{"Content-Type": {"application/json"}}, bytes.NewBufferString(`{"slice": [1, 2, 3]}`), "", baseCtx, `
			method = request.method
			title = request.json_body.slice.foo
		`, http.Header{"method": {http.MethodGet}}},
		{"Variables / GET /w json body & null value", http.MethodGet, http.Header{"Content-Type": {"application/json"}}, bytes.NewBufferString(`{"json": null}`), "", baseCtx, `
			method = request.method
			title = request.json_body.json
		`, http.Header{"method": {http.MethodGet}, "title": nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := test.New(t)

			req := httptest.NewRequest(tt.reqMethod, "https://couper.io/"+tt.query, tt.body)
			*req = *req.Clone(context.WithValue(req.Context(), request.Endpoint, "couper-proxy"))

			for k, v := range tt.reqHeader {
				req.Header[k] = v
			}

			bereq := req.Clone(context.Background())
			beresp := newBeresp(bereq)

			helper.Must(eval.SetGetBody(req, 512))

			ctx := baseCtx.WithClientRequest(req).WithBeresps(beresp).HCLContext()
			ctx.Functions = nil // we are not interested in a functions test

			var resultMap map[string]cty.Value
			err := hclsimple.Decode(tt.name+".hcl", []byte(tt.hcl), ctx, &resultMap)
			// Expect same behaviour as in proxy impl and downgrade missing map elements to warnings.
			if err != nil && seetie.SetSeverityLevel(err.(hcl.Diagnostics)).HasErrors() {
				t.Fatal(err)
			}

			for k, v := range tt.want {
				cv, ok := resultMap[k]
				if !ok {
					t.Errorf("Expected value for %q, got nothing", k)
				}

				result := seetie.ValueToStringSlice(cv)
				if !reflect.DeepEqual(v, result) {
					t.Errorf("Expected %q, got: %#v, type: %#v", v, result, cv)
				}
			}
		})
	}
}

func TestEnvironmentVarMap(t *testing.T) {
	src := []byte(`
server {
	endpoint "/" {
		path = env.COUPER_TEST_VAR_SET
		set_response_headers {
			x-test-value = "${env.COUPER_TEST_VAR_VALUE_SET}"
			x-test-nil = env.COUPER_TEST_VAR_NOT_SET
		}
	}
}
`)
	evalContext := eval.NewContext(src).HCLContext()
	envMap := evalContext.Variables[eval.Environment].AsValueMap()

	if isType := envMap["COUPER_TEST_VAR_SET"].Type(); isType != cty.String {
		t.Errorf("Expected string type for set but empty env, got: %v", isType)
	}

	if isType := envMap["COUPER_TEST_VAR_VALUE_SET"].Type(); isType != cty.String {
		t.Errorf("Expected string type for set non empty env, got: %v", isType)
	}

	if isType := envMap["COUPER_TEST_VAR_NOT_SET"].Type(); isType != cty.String || !envMap["COUPER_TEST_VAR_NOT_SET"].IsNull() {
		t.Errorf("Expected null(string type) for referenced but not set env, got: %v", isType)
	}
}
