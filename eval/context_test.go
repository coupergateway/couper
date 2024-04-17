package eval_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/eval/variables"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/utils"
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

	baseCtx := eval.NewDefaultContext()

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
		t.Run(tt.name, func(subT *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.hcl), "test.hcl", hcl.InitialPos)
			if diags.HasErrors() {
				subT.Fatal(diags)
			}
			bufferOption := buffer.Must(file.Body)

			helper := test.New(subT)

			req := httptest.NewRequest(tt.reqMethod, "https://couper.io/"+tt.query, tt.body)
			ctx := context.WithValue(req.Context(), request.Endpoint, "couper-proxy")
			ctx = context.WithValue(ctx, request.BufferOptions, bufferOption)
			*req = *req.Clone(ctx)

			for k, v := range tt.reqHeader {
				req.Header[k] = v
			}

			bereq := req.Clone(context.Background())
			beresp := newBeresp(bereq)

			helper.Must(eval.SetGetBody(req, buffer.Request, 512))

			ctx, _, _, _ = baseCtx.WithClientRequest(req).WithBeresp(beresp, cty.NilVal)
			hclCtx := ctx.Value(request.ContextType).(*eval.Context).HCLContext()
			hclCtx.Functions = nil // we are not interested in a functions test

			var resultMap map[string]cty.Value
			_ = hclsimple.Decode(tt.name+".hcl", []byte(tt.hcl), hclCtx, &resultMap)

			for k, v := range tt.want {
				cv, ok := resultMap[k]
				if !ok {
					subT.Errorf("Expected value for %q, got nothing", k)
				}

				result := seetie.ValueToStringSlice(cv)
				if !reflect.DeepEqual(v, result) {
					subT.Errorf("Expected %q, got: %#v, type: %#v", v, result, cv)
				}
			}
		})
	}
}

func TestContext_buffer_parseJSON(t *testing.T) {
	baseCtx := eval.NewDefaultContext()
	tests := []struct {
		name                  string
		bufferOptions         buffer.Option
		contentType           string
		expBereqsDefBody      interface{}
		expBereqsDefJsonBody  interface{}
		expBereqBody          interface{}
		expBereqJsonBody      interface{}
		expBerespsDefBody     interface{}
		expBerespsDefJsonBody interface{}
		expBerespBody         interface{}
		expBerespJsonBody     interface{}
	}{
		{
			"buffer both, json-parse both, application/json",
			buffer.Request | buffer.JSONParseRequest | buffer.Response | buffer.JSONParseResponse,
			"application/json",
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
		},
		{
			"buffer both, json-parse both, application/foo+json",
			buffer.Request | buffer.JSONParseRequest | buffer.Response | buffer.JSONParseResponse,
			"application/foo+json",
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
		},
		{
			"buffer req, json-parse req",
			buffer.Request | buffer.JSONParseRequest,
			"application/json",
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			`{"a":"1"}`,
			map[string]interface{}{"a": "1"},
			nil,
			nil,
			nil,
			nil,
		},
		{
			"buffer resp, json-parse resp",
			buffer.Response | buffer.JSONParseResponse,
			"application/json",
			nil,
			nil,
			nil,
			nil,
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
			`{"b":"2"}`,
			map[string]interface{}{"b": "2"},
		},
		{
			"buffer both, don't json-parse",
			buffer.Request | buffer.Response,
			"application/json",
			`{"a":"1"}`,
			map[string]interface{}{},
			`{"a":"1"}`,
			map[string]interface{}{},
			`{"b":"2"}`,
			map[string]interface{}{},
			`{"b":"2"}`,
			map[string]interface{}{},
		},
		{
			"don't buffer, don't json-parse",
			buffer.None,
			"application/json",
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		},
		{
			"don't buffer, json-parse both",
			buffer.None,
			"application/json",
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		},
		{
			"buffer both, json-parse both, text/plain",
			buffer.Request | buffer.JSONParseRequest | buffer.Response | buffer.JSONParseResponse,
			"text/plain",
			`{"a":"1"}`,
			map[string]interface{}{},
			`{"a":"1"}`,
			map[string]interface{}{},
			`{"b":"2"}`,
			map[string]interface{}{},
			`{"b":"2"}`,
			map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)

			req := httptest.NewRequest(http.MethodPost, "/test", io.NopCloser(strings.NewReader(`{"a":"1"}`)))
			req.Header.Set("Content-Type", tt.contentType)
			helper.Must(eval.SetGetBody(req, buffer.Request, 512))

			ctx := context.WithValue(req.Context(), request.BufferOptions, tt.bufferOptions)
			resp := &http.Response{
				Body: io.NopCloser(strings.NewReader(`{"b":"2"}`)),
				Header: http.Header{
					"Content-Type": []string{tt.contentType},
				},
				Request: req.WithContext(ctx),
			}

			ctx, _, _, _ = baseCtx.WithBeresp(resp, cty.NilVal)
			hclContext := ctx.Value(request.ContextType).(*eval.Context).HCLContext()

			beRequests := seetie.ValueToMap(hclContext.Variables[variables.BackendRequests])
			defaultRequest := beRequests["default"].(map[string]interface{})
			if defaultRequest["body"] != tt.expBereqsDefBody {
				subT.Errorf("backend_requests.default.body expected: %#v, got: %#v", tt.expBereqsDefBody, defaultRequest["body"])
			}
			if diff := cmp.Diff(defaultRequest["json_body"], tt.expBereqsDefJsonBody); diff != "" {
				subT.Errorf("backend_requests.default.json_body expected: %#v, got: %#v", tt.expBereqsDefJsonBody, defaultRequest["json_body"])
			}

			beRequest := seetie.ValueToMap(hclContext.Variables[variables.BackendRequest])
			if beRequest["body"] != tt.expBereqBody {
				subT.Errorf("backend_request.body expected: %#v, got: %#v", tt.expBereqBody, beRequest["body"])
			}
			if diff := cmp.Diff(beRequest["json_body"], tt.expBereqJsonBody); diff != "" {
				subT.Errorf("backend_request.json_body expected: %#v, got: %#v", tt.expBereqJsonBody, beRequest["json_body"])
			}

			beResponses := seetie.ValueToMap(hclContext.Variables[variables.BackendResponses])
			defaultResponse := beResponses["default"].(map[string]interface{})
			if defaultResponse["body"] != tt.expBerespsDefBody {
				subT.Errorf("backend_responses.default.body expected: %#v, got: %#v", tt.expBerespsDefBody, defaultResponse["body"])
			}
			if diff := cmp.Diff(defaultResponse["json_body"], tt.expBerespsDefJsonBody); diff != "" {
				subT.Errorf("backend_responses.default.json_body expected: %#v, got: %#v", tt.expBerespsDefJsonBody, defaultResponse["json_body"])
			}

			beResponse := seetie.ValueToMap(hclContext.Variables[variables.BackendResponse])
			if beResponse["body"] != tt.expBerespBody {
				subT.Errorf("backend_response.body expected: %#v, got: %#v", tt.expBerespBody, beResponse["body"])
			}
			if diff := cmp.Diff(beResponse["json_body"], tt.expBerespJsonBody); diff != "" {
				subT.Errorf("backend_response.json_body expected: %#v, got: %#v", tt.expBerespJsonBody, beResponse["json_body"])
			}
		})
	}
}

func TestDefaultEnvVariables(t *testing.T) {
	tests := []struct {
		name string
		hcl  string
		want map[string]cty.Value
	}{
		{
			"test",
			`
			server "test" {
			  endpoint "/" {
				proxy {
				  backend {
					origin = env.ORIGIN
					timeout = env.TIMEOUT
				  }
				}
			  }
			}

			defaults {
				environment_variables = {
					ORIGIN = "FOO"
					TIMEOUT = "42"
					IGNORED = "bar"
				}
			}
			`,
			map[string]cty.Value{"ORIGIN": cty.StringVal("FOO"), "TIMEOUT": cty.StringVal("42")},
		},
		{
			"no-environment_variables-block",
			`
			server "test" {
			  endpoint "/" {
				proxy {
				  backend {
					origin = env.ORIGIN
					timeout = env.TIMEOUT
				  }
				}
			  }
			}

			defaults {}
			`,
			map[string]cty.Value{"ORIGIN": cty.StringVal(""), "TIMEOUT": cty.StringVal("")},
		},
		{
			"no-defaults-block",
			`
			server "test" {
			  endpoint "/" {
				proxy {
				  backend {
					origin = env.ORIGIN
					timeout = env.TIMEOUT
				  }
				}
			  }
			}
			`,
			map[string]cty.Value{"ORIGIN": cty.StringVal(""), "TIMEOUT": cty.StringVal("")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				subT.Error(err)
				return
			}

			hclContext := cf.Context.(*eval.Context).HCLContext()

			envVars := hclContext.Variables["env"].AsValueMap()
			for key, expectedValue := range tt.want {
				value, isset := envVars[key]
				if !isset && expectedValue != cty.NilVal {
					subT.Errorf("Missing evironment variable %q:\nWant:\t%s=%q\nGot:", key, key, expectedValue)
				} else if value != expectedValue {
					subT.Errorf("Unexpected value for evironment variable %q:\nWant:\t%s=%q\nGot:\t%s=%q", key, key, expectedValue, key, value)
				}
			}
		})
	}
}

func TestCouperVariables(t *testing.T) {
	tests := []struct {
		name string
		hcl  string
		env  string
		want map[string]string
	}{
		{
			"test",
			`
			server "test" {}
			`,
			"",
			map[string]string{"version": utils.VersionName, "environment": ""},
		},
		{
			"environment",
			`
			server {}
			`,
			"bar",
			map[string]string{"version": utils.VersionName, "environment": "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			cf, err := configload.LoadBytesEnv([]byte(tt.hcl), "couper.hcl", tt.env)
			if err != nil {
				subT.Error(err)
				return
			}

			hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

			couperVars := seetie.ValueToMap(hclContext.Variables["couper"])

			if len(couperVars) != len(tt.want) {
				subT.Errorf("Unexpected 'couper' variables:\nWant:\t%q\nGot:\t%q", tt.want, couperVars)
			}
			for key, expectedValue := range tt.want {
				value := couperVars[key]
				if value != expectedValue {
					subT.Errorf("Unexpected value for variable:\nWant:\tcouper.%s=%q\nGot:\tcouper.%s=%q", key, expectedValue, key, value)
				}
			}
		})
	}
}
