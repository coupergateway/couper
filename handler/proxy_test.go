package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestProxy_ServeHTTP_Timings(t *testing.T) {
	t.Skip("todo: fixme")
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb proxy settings
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name           string
		options        *handler.ProxyOptions
		req            *http.Request
		expectedStatus int
	}{
		{"with zero timings", &handler.ProxyOptions{Context: test.NewRemainContext("origin", origin.URL)}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusNoContent},
		{"with overall timeout", &handler.ProxyOptions{Context: test.NewRemainContext("origin", "http://1.2.3.4/"), Timeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusBadGateway},
		{"with connect timeout", &handler.ProxyOptions{Context: test.NewRemainContext("origin", "http://blackhole.webpagetest.org/"), ConnectTimeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusBadGateway},
		{"with ttfb timeout", &handler.ProxyOptions{Context: test.NewRemainContext("origin", origin.URL), TTFBTimeout: time.Second}, httptest.NewRequest(http.MethodHead, "http://1.2.3.4/", nil), http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := handler.NewProxy(tt.options, logger.WithContext(nil), nil, eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectedStatus, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					t.Error(log.Message)
				}
			}
		})
	}
}

func TestProxy_ServeHTTP_CORS_PFC(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("from upstream"))
	}))
	defer origin.Close()

	tests := []struct {
		name                    string
		corsOptions             *handler.CORSOptions
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{
			"with ACRM",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRH",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                         "https://www.example.com",
				"Access-Control-Request-Headers": "X-Foo, X-Bar",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "X-Foo, X-Bar",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, ACRH",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                         "https://www.example.com",
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "X-Foo, X-Bar",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "X-Foo, X-Bar",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, max-age",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, MaxAge: "3600"},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "3600",
			},
		},
		{
			"origin mismatch",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                        "https://www.example.org",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := handler.NewProxy(&handler.ProxyOptions{Context: test.NewRemainContext("origin", origin.URL), CORS: tt.corsOptions}, logger.WithContext(nil), nil, eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)

			tt.expectedResponseHeaders["Vary"] = ""
			tt.expectedResponseHeaders["Content-Type"] = ""

			for name, expValue := range tt.expectedResponseHeaders {
				value := rec.HeaderMap.Get(name)
				if value != expValue {
					t.Errorf("Expected %s %s, got: %s", name, expValue, value)
				}
			}

			if rec.Code != http.StatusNoContent {
				t.Errorf("Expected status %d, got: %d", http.StatusNoContent, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					t.Error(log.Message)
				}
			}
		})
	}
}

func TestProxy_ServeHTTP_CORS(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("from upstream"))
	}))
	defer origin.Close()

	tests := []struct {
		name                    string
		corsOptions             *handler.CORSOptions
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{
			"specific origin",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "Origin",
			},
		},
		{
			"specific origins",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com", "https://example.com"}},
			map[string]string{
				"Origin": "https://example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://example.com",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "Origin",
			},
		},
		{
			"any origin",
			&handler.CORSOptions{AllowedOrigins: []string{"*"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "",
			},
		},
		{
			"any and specific origin",
			&handler.CORSOptions{AllowedOrigins: []string{"https://example.com", "https://www.example.com", "*"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "",
			},
		},
		{
			"specific origin, cookie credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin": "https://www.example.com",
				"Cookie": "a=b",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			"specific origin, auth credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin":        "https://www.example.com",
				"Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			"any origin, cookie credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin": "https://www.example.com",
				"Cookie": "a=b",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
		{
			"any origin, auth credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin":        "https://www.example.com",
				"Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
		{
			"any origin, proxy auth credentials",
			&handler.CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin":              "https://www.example.com",
				"Proxy-Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := handler.NewProxy(&handler.ProxyOptions{Context: test.NewRemainContext("origin", origin.URL), CORS: tt.corsOptions}, logger.WithContext(context.Background()), nil, eval.NewENVContext(nil))
			if err != nil {
				subT.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)

			for name, expValue := range tt.expectedResponseHeaders {
				value := rec.HeaderMap.Get(name)
				if value != expValue {
					subT.Errorf("%s:\n\tExpected: %s %q, got: %s", tt.name, name, expValue, value)
				}
			}

			if rec.Code != http.StatusOK {
				subT.Errorf("Expected status %d, got: %d", http.StatusOK, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					subT.Error(log.Message)
				}
			}
		})
	}
}

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

			p, err := handler.NewProxy(&handler.ProxyOptions{
				Context: hclContext,
				CORS:    &handler.CORSOptions{},
			}, nullLog, nil, eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			*req = *req.Clone(tt.ctx)

			proxy := p.(*handler.Proxy)
			err = proxy.Director(req)
			if err != nil {
				t.Fatal(err)
			}

			attr, _ := hclContext[0].JustAttributes()
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
	type fields struct {
		evalContext *hcl.EvalContext
		options     *handler.ProxyOptions
	}

	type header map[string]string

	type testCase struct {
		name       string
		fields     fields
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

	opts := &handler.ProxyOptions{
		BackendName:      "test-origin",
		Context:          test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
		CORS:             &handler.CORSOptions{},
		RequestBodyLimit: 10,
	}

	tests := []testCase{
		{"GET use req.Header", fields{baseCtx, opts}, `
		set_response_headers = {
			X-Method = req.method
		}`, http.MethodGet, nil, header{"X-Method": http.MethodGet}, false},
		{"POST use req.post", fields{baseCtx, opts}, `
		set_response_headers = {
			X-Method = req.method
			X-Post = req.post.foo
		}`, http.MethodPost, strings.NewReader(`foo=bar`), header{
			"X-Method": http.MethodPost,
			"X-Post":   "bar",
		}, false},
	}

	for _, tt := range tests {
		hook.Reset()
		t.Run(tt.name, func(t *testing.T) {
			var remain hclBody
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), baseCtx, &remain)
			if err != nil {
				t.Fatal(err)
			}
			tt.fields.options.Context = append(tt.fields.options.Context, remain.Inline)
			p, err := handler.NewProxy(tt.fields.options, log.WithContext(context.Background()), nil, tt.fields.evalContext)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(tt.method, "http://couper.io", tt.body)
			rw := httptest.NewRecorder()

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			p.ServeHTTP(rw, req)
			res := rw.Result()

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

func TestProxy_SetGetBody_LimitBody_Roundtrip(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		name    string
		limit   int64
		payload string
		wantErr error
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", 12, "content", nil},
		{"/w zero limit", 0, "01", errors.APIReqBodySizeExceeded},
		{"/w limit /w oversize body", 4, "12345", errors.APIReqBodySizeExceeded},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			proxy, _, _, closeFn := helper.NewProxy(&handler.ProxyOptions{
				Context:          helper.NewProxyContext("set_request_headers = { x = req.post }"), // ensure buffering is enabled
				RequestBodyLimit: testcase.limit,
			})
			closeFn() // unused

			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))
			err := proxy.SetGetBody(req)
			if !reflect.DeepEqual(err, testcase.wantErr) {
				subT.Errorf("Expected '%v', got: '%v'", testcase.wantErr, err)
			}
		})
	}
}

func Test_IsCredentialed(t *testing.T) {
	type testCase struct {
		name           string
		requestHeaders map[string]string
		exp            bool
	}

	tests := []testCase{
		{
			"Cookie",
			map[string]string{"Cookie": "a=b"},
			true,
		},
		{
			"Authorization",
			map[string]string{"Authorization": "Basic qeinbqtpoib"},
			true,
		},
		{
			"Proxy-Authorization",
			map[string]string{"Proxy-Authorization": "Basic qeinbqtpoib"},
			true,
		},
		{
			"Not credentialed",
			map[string]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			credentialed := handler.IsCredentialed(req.Header)
			if credentialed != tt.exp {
				t.Errorf("expected: %t, got: %t", tt.exp, credentialed)
			}
		})
	}
}

func TestProxy_setRoundtripContext_Variables_json_body(t *testing.T) {
	type want struct {
		req test.Header
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
		}`, []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
		}, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": "bar"}}},
		{"method /w body", `
		origin = "http://1.2.3.4/"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, []string{http.MethodTrace}, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": ""}}},
		{"method /wo body", `
		origin = "http://1.2.3.4/"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		}, test.Header{"Content-Type": "application/json"}, "", want{req: test.Header{"x-test": ""}}},
	}

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(method+" "+tt.name, func(subT *testing.T) {
				helper := test.New(subT)
				proxy, _, _, closeFn := helper.NewProxy(&handler.ProxyOptions{
					Context:          helper.NewProxyContext(tt.inlineCtx),
					CORS:             &handler.CORSOptions{},
					RequestBodyLimit: 64,
				})

				closeFn() // unused

				var body io.Reader
				if tt.body != "" {
					body = bytes.NewBufferString(tt.body)
				}
				req := httptest.NewRequest(method, "/", body)
				tt.header.Set(req)
				helper.Must(proxy.Director(req))

				for k, v := range tt.want.req {
					if req.Header.Get(k) != v {
						subT.Errorf("want: %q for key %q, got: %q", v, k, req.Header.Get(k))
					}
				}
			})
		}
	}
}

func TestServer_DisableCompression(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Accept-Encoding") != "" {
			t.Error("Unexpected Accept-Encoding header")
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()

	var hclBody []hcl.Body
	u := seetie.GoToValue(origin.URL)
	hclBody = append(hclBody, hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	}))

	p, err := handler.NewProxy(
		&handler.ProxyOptions{Context: hclBody},
		logger.WithContext(nil),
		nil, eval.NewENVContext(nil),
	)
	helper.Must(err)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	p.ServeHTTP(httptest.NewRecorder(), req)
}

func TestServer_ModifyAcceptEncoding(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if ae := req.Header.Get("Accept-Encoding"); ae != "gzip" {
			t.Errorf("Unexpected Accept-Encoding header: %s", ae)
		}

		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		for i := 1; i < 1000; i++ {
			w.Write([]byte("<html/>"))
		}
		w.Close()

		rw.Header().Set("Content-Encoding", "gzip")
		rw.Write(b.Bytes())
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()

	var hclBody []hcl.Body
	u := seetie.GoToValue(origin.URL)
	hclBody = append(hclBody, hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	}))

	p, err := handler.NewProxy(
		&handler.ProxyOptions{Context: hclBody},
		logger.WithContext(nil),
		nil, eval.NewENVContext(nil),
	)
	helper.Must(err)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	res := httptest.NewRecorder()
	p.ServeHTTP(res, req)

	if l := res.Header().Get("Content-Length"); l != "60" {
		t.Errorf("Unexpected C/L: %s", l)
	}
	if l := len(res.Body.Bytes()); l != 6993 {
		t.Errorf("Unexpected body length: %d", l)
	}
}
