package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestProxy_director(t *testing.T) {
	helper := test.New(t)

	log, _ := logrustest.NewNullLogger()
	nullLog := log.WithContext(nil)

	bgCtx := context.Background()

	tests := []struct {
		name      string
		inlineCtx string
		path      string
		ctx       context.Context
		expReq    *http.Request
	}{
		{"proxy url settings", `origin = "http://1.2.3.4"`, "", bgCtx, httptest.NewRequest("GET", "http://1.2.3.4", nil)},
		{"proxy url settings w/hostname", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
		`, "", bgCtx, httptest.NewRequest("GET", "http://couper.io", nil)},
		{"proxy url settings w/wildcard ctx", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/**"
		`, "/peter", context.WithValue(bgCtx, request.Wildcard, "/hans"), httptest.NewRequest("GET", "http://couper.io/hans", nil)},
		{"proxy url settings w/wildcard ctx empty", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/docs/**"
		`, "", context.WithValue(bgCtx, request.Wildcard, ""), httptest.NewRequest("GET", "http://couper.io/docs", nil)},
		{"proxy url settings w/wildcard ctx empty /w trailing path slash", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/docs/**"
		`, "/", context.WithValue(bgCtx, request.Wildcard, ""), httptest.NewRequest("GET", "http://couper.io/docs/", nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hclContext := helper.NewProxyContext(tt.inlineCtx)

			backend := transport.NewBackend(eval.NewENVContext(nil), hcl.EmptyBody(), &transport.Config{}, nullLog, nil)
			proxy := handler.NewProxy(backend, hclContext, eval.NewENVContext(nil))

			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			*req = *req.Clone(tt.ctx)

			_, _ = proxy.RoundTrip(req) // implicit director() call

			attr, _ := hclContext.JustAttributes()
			hostnameExp, ok := attr["hostname"]

			if !ok && tt.expReq.Host != req.Host {
				t.Errorf("expected same host value, want: %q, got: %q", req.Host, tt.expReq.Host)
			} else if ok {
				hostVal, _ := hostnameExp.Expr.Value(eval.NewENVContext(nil))
				hostname := seetie.ValueToString(hostVal)
				if hostname != tt.expReq.Host {
					t.Errorf("expected a configured request host: %q, got: %q", hostname, tt.expReq.Host)
				}
			}

			if req.URL.Path != tt.expReq.URL.Path {
				t.Errorf("expected path: %q, got: %q", tt.expReq.URL.Path, req.URL.Path)
			}
		})
	}
}

func TestProxy_ServeHTTP_Eval(t *testing.T) {
	type header map[string]string

	type testCase struct {
		name       string
		hcl        string
		method     string
		body       io.Reader
		wantHeader header
		wantErr    bool
	}

	baseCtx := eval.NewENVContext(nil)
	log, hook := logrustest.NewNullLogger()

	type hclBody struct {
		Inline hcl.Body `hcl:",remain"`
	}

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
		}

		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []testCase{
		{"GET use req.Header", `
		set_response_headers = {
			X-Method = req.method
		}`, http.MethodGet, nil, header{"X-Method": http.MethodGet}, false},
		{"POST use req.post", `
		set_response_headers = {
			X-Method = req.method
			X-Post = req.post.foo
		}`, http.MethodPost, strings.NewReader(`foo=bar`), header{
			"X-Method": http.MethodPost,
			"X-Post":   "bar",
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := test.New(t)
			hook.Reset()

			var remain hclBody
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), baseCtx, &remain)
			helper.Must(err)

			l := log.WithContext(context.Background())
			evalCtx := eval.NewENVContext(nil)
			backend := transport.NewBackend(evalCtx, hcl.EmptyBody(), &transport.Config{}, l, nil)
			proxy := handler.NewProxy(backend, configload.MergeBodies([]hcl.Body{
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				remain.Inline,
			}), evalCtx)

			req := httptest.NewRequest(tt.method, "http://couper.io", tt.body)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			res, err := proxy.RoundTrip(req)
			if err != nil {
				t.Error(err)
			}

			if res == nil {
				t.Fatal("Expected a response")
				t.Log(hook.LastEntry().String())
			}

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("Expected StatusNoContent 204, got: %q %d", res.Status, res.StatusCode)
				t.Log(hook.LastEntry().String())
			}

			for k, v := range tt.wantHeader {
				if got := res.Header.Get(k); got != v {
					t.Errorf("Expected value for header %q: %q, got: %q", k, v, got)
					t.Log(hook.LastEntry().String())
				}
			}

		})
	}
}

func TestProxy_setRoundtripContext_Variables_json_body(t *testing.T) {
	type want struct {
		req test.Header
	}

	defaultMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
	}

	tests := []struct {
		name      string
		inlineCtx string
		methods   []string
		header    test.Header
		body      string
		want      want
	}{
		{"method /w body", `
		origin = "http://1.2.3.4/"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, defaultMethods, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": "bar"}}},
		{"method /w body", `
		origin = "http://1.2.3.4/"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, []string{http.MethodTrace}, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": ""}}},
		{"method /wo body", `
		origin = "http://1.2.3.4/"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, append(defaultMethods, http.MethodTrace),
			test.Header{"Content-Type": "application/json"}, "", want{req: test.Header{"x-test": ""}}},
	}

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(method+" "+tt.name, func(subT *testing.T) {
				helper := test.New(subT)
				proxy := helper.NewProxy(&transport.Config{}, hcl.EmptyBody(), helper.NewProxyContext(tt.inlineCtx))

				var body io.Reader
				if tt.body != "" {
					body = bytes.NewBufferString(tt.body)
				}
				req := httptest.NewRequest(method, "/", body)
				tt.header.Set(req)

				_, _ = proxy.RoundTrip(req) // indirect director call

				for k, v := range tt.want.req {
					if req.Header.Get(k) != v {
						subT.Errorf("want: %q for key %q, got: %q", v, k, req.Header.Get(k))
					}
				}
			})
		}
	}
}

// TestProxy_SetRoundtripContext_Null_Eval tests the handling with non existing references or cty.Null evaluations.
func TestProxy_SetRoundtripContext_Null_Eval(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		name       string
		remain     string
		expHeaders test.Header
	}

	clientPayload := []byte(`{ "client": true, "origin": false, "nil": null }`)
	originPayload := []byte(`{ "client": false, "origin": true, "nil": null }`)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientData, err := ioutil.ReadAll(r.Body)
		helper.Must(err)
		if !bytes.Equal(clientData, clientPayload) {
			t.Errorf("Expected a request with client payload, got %q", string(clientData))
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_, err = rw.Write(originPayload)
		helper.Must(err)
	}))

	for i, tc := range []testCase{
		{"no eval", `path = "/"`, test.Header{}},
		{"json_body client field", `set_response_headers = { "x-client" = "my-val-x-${req.json_body.client}" }`,
			test.Header{
				"x-client": "my-val-x-true",
			}},
		{"json_body non existing field", `set_response_headers = {
"${beresp.json_body.not-there}" = "my-val-0-${beresp.json_body.origin}"
"${req.json_body.client}-my-val-a" = "my-val-b-${beresp.json_body.client}"
}`,
			test.Header{"true-my-val-a": ""}}, // since one reference is failing ('not-there') the whole block does
		{"json_body null value", `set_response_headers = { "x-null" = "${beresp.json_body.nil}" }`, test.Header{"x-null": ""}},
	} {
		t.Run(tc.name, func(st *testing.T) {
			h := test.New(st)

			proxy := h.NewProxy(nil, hcl.EmptyBody(), configload.MergeBodies([]hcl.Body{
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				helper.NewProxyContext(tc.remain),
			}))

			req := httptest.NewRequest(http.MethodGet, "http://localhost/", bytes.NewReader(clientPayload))
			req.Header.Set("Content-Type", "application/json")
			*req = *req.WithContext(context.WithValue(req.Context(), request.UID, fmt.Sprintf("#%.2d: %s", i+1, tc.name)))

			res, err := proxy.RoundTrip(req)
			h.Must(err)

			if res.StatusCode != http.StatusOK {
				st.Errorf("Expected StatusOK, got: %d", res.StatusCode)
			}

			originData, err := ioutil.ReadAll(res.Body)
			h.Must(err)

			if !bytes.Equal(originPayload, originData) {
				st.Errorf("Expected same origin payload, got:\n%s\nlog message:\n", string(originData))
			}

			for k, v := range tc.expHeaders {
				if res.Header.Get(k) != v {
					t.Errorf("Expected header %q value: %q, got: %q", k, v, res.Header.Get(k))
				}
			}
		})

	}
}

// TestProxy_BufferingOptions tests the option interaction with enabled/disabled validation and
// the requirement for buffering to read the post or json body.
func TestProxy_BufferingOptions(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		name           string
		apiOptions     *validation.OpenAPIOptions
		remain         string
		expectedOption eval.BufferOption
	}

	clientPayload := []byte(`{ "client": true, "origin": false }`)
	originPayload := []byte(`{ "client": false, "origin": true }`)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientData, err := ioutil.ReadAll(r.Body)
		helper.Must(err)
		if !bytes.Equal(clientData, clientPayload) {
			t.Errorf("Expected a request with client payload, got %q", string(clientData))
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_, err = rw.Write(originPayload)
		helper.Must(err)
	}))

	newOptions := func() *validation.OpenAPIOptions {
		c := config.OpenAPI{}
		conf, err := validation.NewOpenAPIOptionsFromBytes(&c, helper.NewOpenAPIConf("/"))
		helper.Must(err)
		return conf
	}

	for i, tc := range []testCase{
		{"no buffering", nil, `path = "/"`, eval.BufferNone},
		{"req buffer json.body", nil, `path = "/${req.json_body.client}"`, eval.BufferRequest},
		{"beresp buffer json.body", nil, `response_headers = { x-test = "${beresp.json_body.origin}" }`, eval.BufferResponse},
		{"bereq/beresp validation", newOptions(), `path = "/"`, eval.BufferRequest | eval.BufferResponse},
		{"beresp validation", newOptions(), `path = "/"`, eval.BufferResponse},
		{"bereq validation", newOptions(), `path = "/"`, eval.BufferRequest},
		{"no validation", newOptions(), `path = "/"`, eval.BufferNone},
		{"req buffer json.body & beresp validation", newOptions(), `set_response_headers = { x-test = "${req.json_body.client}" }`, eval.BufferRequest | eval.BufferResponse},
		{"beresp buffer json.body & bereq validation", newOptions(), `set_response_headers = { x-test = "${beresp.json_body.origin}" }`, eval.BufferRequest | eval.BufferResponse},
		{"req buffer json.body & bereq validation", newOptions(), `set_response_headers = { x-test = "${req.json_body.client}" }`, eval.BufferRequest},
		{"beresp buffer json.body & beresp validation", newOptions(), `set_response_headers = { x-test = "${beresp.json_body.origin}" }`, eval.BufferResponse},
	} {
		t.Run(tc.name, func(st *testing.T) {
			h := test.New(st)

			proxy := h.NewProxy(nil, hcl.EmptyBody(), configload.MergeBodies([]hcl.Body{
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				helper.NewProxyContext(tc.remain),
			}))

			configuredOption := reflect.ValueOf(proxy).Elem().FieldByName("bufferOption") // private field: ro
			opt := eval.BufferOption(configuredOption.Uint())
			if (opt & tc.expectedOption) != tc.expectedOption {
				st.Errorf("Expected option: %#v, got: %#v", tc.expectedOption, opt)
			}

			req := httptest.NewRequest(http.MethodGet, "http://localhost/", bytes.NewReader(clientPayload))
			req.Header.Set("Content-Type", "application/json")
			*req = *req.WithContext(context.WithValue(req.Context(), request.UID, fmt.Sprintf("#%.2d: %s", i+1, tc.name)))

			res, err := proxy.RoundTrip(req)
			h.Must(err)

			if res.StatusCode != http.StatusOK {
				st.Errorf("Expected StatusOK, got: %d", res.StatusCode)
			}

			originData, err := ioutil.ReadAll(res.Body)
			h.Must(err)

			if !bytes.Equal(originPayload, originData) {
				st.Errorf("Expected same origin payload, got:\n%s\nlog message:\n", string(originData))
			}
		})

	}
}
