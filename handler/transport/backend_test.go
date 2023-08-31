package transport_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config"
	hclbody "github.com/coupergateway/couper/config/body"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/handler/transport"
	"github.com/coupergateway/couper/handler/validation"
	"github.com/coupergateway/couper/internal/seetie"
	"github.com/coupergateway/couper/internal/test"
)

func TestBackend_RoundTrip_Timings(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb and overall timeout
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	withTimingsFn := func(base *hclsyntax.Body, connect, ttfb, timeout string) *hclsyntax.Body {
		content := &hclsyntax.Body{Attributes: hclsyntax.Attributes{
			"connect_timeout": {Name: "connect_timeout", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(connect)}},
			"ttfb_timeout":    {Name: "ttfb_timeout", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(ttfb)}},
			"timeout":         {Name: "timeout", Expr: &hclsyntax.LiteralValueExpr{Val: cty.StringVal(timeout)}},
		}}
		return hclbody.MergeBodies(base, content, true)
	}

	tests := []struct {
		name        string
		context     *hclsyntax.Body
		req         *http.Request
		expectedErr string
	}{
		{"with zero timings", hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL), httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), ""},
		{"with overall timeout", withTimingsFn(hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL), "1m", "30s", "500ms"), httptest.NewRequest(http.MethodHead, "http://1.2.3.5/", nil), "deadline exceeded"},
		{"with connect timeout", withTimingsFn(hclbody.NewHCLSyntaxBodyWithStringAttr("origin", "http://blackhole.webpagetest.org"), "750ms", "500ms", "1m"), httptest.NewRequest(http.MethodGet, "http://1.2.3.6/", nil), "i/o timeout"},
		{"with ttfb timeout", withTimingsFn(hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL), "10s", "1s", "1m"), httptest.NewRequest(http.MethodHead, "http://1.2.3.7/", nil), "timeout awaiting response headers"},
	}

	logger, hook := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			backend := transport.NewBackend(tt.context, &transport.Config{NoProxyFromEnv: true}, nil, log)

			_, err := backend.RoundTrip(tt.req)
			if err != nil && tt.expectedErr == "" {
				subT.Error(err)
				return
			}

			gerr, isErr := err.(errors.GoError)

			if tt.expectedErr != "" &&
				(err == nil || !isErr || !strings.HasSuffix(gerr.LogError(), tt.expectedErr)) {
				subT.Errorf("Expected err %s, got: %#v", tt.expectedErr, err)
			}
		})
	}
}

func TestBackend_Compression_Disabled(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Accept-Encoding") != "" {
			t.Error("Unexpected Accept-Encoding header")
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	hclBody := hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL)
	backend := transport.NewBackend(hclBody, &transport.Config{}, nil, log)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204, got: %d", res.StatusCode)
	}
}

func TestBackend_Compression_ModifyAcceptEncoding(t *testing.T) {
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
	log := logger.WithContext(context.Background())

	hclBody := hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL)

	backend := transport.NewBackend(hclBody, &transport.Config{
		Origin: origin.URL,
	}, nil, log)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	req = req.WithContext(context.WithValue(context.Background(), request.BufferOptions, eval.BufferResponse))
	req.Header.Set("Accept-Encoding", "br, gzip")
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if res.ContentLength != 60 {
		t.Errorf("Unexpected C/L: %d", res.ContentLength)
	}

	n, err := io.Copy(io.Discard, res.Body)
	helper.Must(err)

	if n != 6993 {
		t.Errorf("Unexpected body length: %d, want: %d", n, 6993)
	}
}

func TestBackend_RoundTrip_Validation(t *testing.T) {
	helper := test.New(t)
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		if req.URL.RawQuery == "404" {
			rw.WriteHeader(http.StatusNotFound)
		}
		_, err := rw.Write([]byte("from upstream"))
		helper.Must(err)
	}))
	defer origin.Close()

	openAPIYAML := helper.NewOpenAPIConf("/pa/./th")

	tests := []struct {
		name               string
		openapi            *config.OpenAPI
		requestMethod      string
		requestPath        string
		expectedErr        string
		expectedLogMessage string
	}{
		{
			"valid request / valid response",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/pa/./th",
			"",
			"",
		},
		{
			"valid path: trailing /",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodPost,
			"/pa/./th/",
			"backend error",
			"",
		},
		{
			"invalid path: no path resolution",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/pa/th",
			"backend error",
			"'GET /pa/th': no matching operation was found",
		},
		{
			"invalid path: double /",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/pa/.//th",
			"backend error",
			"'GET /pa/.//th': no matching operation was found",
		},
		{ // gorilla/mux router has .UseEncodedPath(), see https://pkg.go.dev/github.com/gorilla/mux#Router.UseEncodedPath
			"URL encoded request",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/pa%2f%2e%2fth",
			"backend error",
			"'GET /pa/./th': no matching operation was found",
		},
		{ // gorilla/mux router has .UseEncodedPath(), see https://pkg.go.dev/github.com/gorilla/mux#Router.UseEncodedPath
			"URL encoded request, wrong method",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodPost,
			"/pa%2f%2e%2fth",
			"backend error",
			"'POST /pa/./th': no matching operation was found",
		},
		{
			"invalid request",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodPost,
			"/pa/./th",
			"backend error",
			"'POST /pa/./th': method not allowed",
		},
		{
			"invalid request, IgnoreRequestViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreRequestViolations: true, IgnoreResponseViolations: true},
			http.MethodPost,
			"/pa/./th",
			"",
			"'POST /pa/./th': method not allowed",
		},
		{
			"invalid response",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/pa/./th?404",
			"backend error",
			"status is not supported",
		},
		{
			"invalid response, IgnoreResponseViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreResponseViolations: true},
			http.MethodGet,
			"/pa/./th?404",
			"",
			"status is not supported",
		},
	}

	logger, hook := test.NewLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			openapiValidatorOptions, err := validation.NewOpenAPIOptionsFromBytes(tt.openapi, openAPIYAML)
			if err != nil {
				subT.Fatal(err)
			}
			content := helper.NewInlineContext(`
				origin = "` + origin.URL + `"
			`)

			backend := transport.NewBackend(content, &transport.Config{}, &transport.BackendOptions{
				OpenAPI: openapiValidatorOptions,
			}, log)

			req := httptest.NewRequest(tt.requestMethod, "http://1.2.3.4"+tt.requestPath, nil)

			_, err = backend.RoundTrip(req)
			if err != nil && tt.expectedErr == "" {
				subT.Fatal(err)
			}

			if tt.expectedErr != "" && (err == nil || err.Error() != tt.expectedErr) {
				subT.Errorf("\nwant:\t%s\ngot:\t%v", tt.expectedErr, err)
				subT.Log(hook.LastEntry().Message)
			}

			entry := hook.LastEntry()
			if tt.expectedLogMessage != "" {
				if data, ok := entry.Data["validation"]; ok {
					var found bool

					for _, errStr := range data.([]string) {
						if errStr != tt.expectedLogMessage {
							subT.Fatalf("\nwant:\t%s\ngot:\t%v", tt.expectedLogMessage, errStr)
						} else {
							found = true
							break
						}
					}

					if !found {
						for _, errStr := range data.([]string) {
							subT.Log(errStr)
						}
						subT.Errorf("expected matching validation error logs:\n\t%s\n\tgot: nothing", tt.expectedLogMessage)
					}
				}
			}
		})
	}
}

func TestBackend_director(t *testing.T) {
	helper := test.New(t)

	log, _ := logrustest.NewNullLogger()
	nullLog := log.WithContext(context.TODO())

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
		t.Run(tt.name, func(subT *testing.T) {
			hclContext := helper.NewInlineContext(tt.inlineCtx)

			backend := transport.NewBackend(hclbody.MergeBodies(hclContext,
				hclbody.NewHCLSyntaxBodyWithStringAttr("timeout", "1s"),
				true,
			), &transport.Config{}, nil, nullLog)

			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			*req = *req.WithContext(tt.ctx)

			beresp, _ := backend.RoundTrip(req) // implicit director()
			// outreq gets set on error cases
			outreq := beresp.Request

			attr, _ := hclContext.JustAttributes()
			hostnameExp, ok := attr["hostname"]

			if !ok && tt.expReq.Host != outreq.Host {
				subT.Errorf("expected same host value, want: %q, got: %q", outreq.Host, tt.expReq.Host)
			} else if ok {
				hostVal, _ := hostnameExp.Expr.Value(eval.NewDefaultContext().HCLContext())
				hostname := seetie.ValueToString(hostVal)
				if hostname != tt.expReq.Host {
					subT.Errorf("expected a configured request host: %q, got: %q", hostname, tt.expReq.Host)
				}
			}

			if outreq.URL.Path != tt.expReq.URL.Path {
				subT.Errorf("expected path: %q, got: %q", tt.expReq.URL.Path, outreq.URL.Path)
			}
		})
	}
}

func TestBackend_HealthCheck(t *testing.T) {
	type testCase struct {
		name        string
		health      *config.Health
		expectation config.HealthCheck
	}

	defaultExpectedStatus := map[int]bool{200: true, 204: true, 301: true}

	toPtr := func(n uint) *uint { return &n }

	for _, tc := range []testCase{
		{
			name:   "health check with default values",
			health: &config.Health{},
			expectation: config.HealthCheck{
				FailureThreshold: 2,
				Interval:         time.Second,
				Timeout:          time.Second,
				ExpectedStatus:   defaultExpectedStatus,
				ExpectedText:     "",
				RequestUIDFormat: "common",
			},
		},
		{
			name: "health check with configured values",
			health: &config.Health{
				FailureThreshold: toPtr(42),
				Interval:         "1h",
				Timeout:          "9m",
				Path:             "/gsund??",
				ExpectedStatus:   []int{418},
				ExpectedText:     "roger roger",
			},
			expectation: config.HealthCheck{
				FailureThreshold: 42,
				Interval:         time.Hour,
				Timeout:          9 * time.Minute,
				ExpectedStatus:   map[int]bool{418: true},
				ExpectedText:     "roger roger",
				Request: &http.Request{URL: &url.URL{
					Scheme:   "http",
					Host:     "origin:8080",
					Path:     "/gsund",
					RawQuery: "?",
				}},
				RequestUIDFormat: "common",
			},
		},
		{
			name:   "uninitialised health check",
			health: nil,
			expectation: config.HealthCheck{
				FailureThreshold: 2,
				Interval:         time.Second,
				Timeout:          time.Second,
				ExpectedStatus:   defaultExpectedStatus,
				ExpectedText:     "",
				RequestUIDFormat: "common",
			},
		},
		{
			name: "timeout set indirectly by configured interval",
			health: &config.Health{
				Interval: "10s",
			},
			expectation: config.HealthCheck{
				FailureThreshold: 2,
				Interval:         10 * time.Second,
				Timeout:          10 * time.Second,
				ExpectedStatus:   defaultExpectedStatus,
				ExpectedText:     "",
				RequestUIDFormat: "common",
			},
		},
		{
			name: "timeout bounded by configured interval",
			health: &config.Health{
				Interval: "5s",
				Timeout:  "10s",
			},
			expectation: config.HealthCheck{
				FailureThreshold: 2,
				Interval:         5 * time.Second,
				Timeout:          5 * time.Second,
				ExpectedStatus:   defaultExpectedStatus,
				ExpectedText:     "",
				RequestUIDFormat: "common",
			},
		},
		{
			name: "zero threshold",
			health: &config.Health{
				FailureThreshold: toPtr(0),
			},
			expectation: config.HealthCheck{
				FailureThreshold: 0,
				Interval:         time.Second,
				Timeout:          time.Second,
				ExpectedStatus:   defaultExpectedStatus,
				ExpectedText:     "",
				RequestUIDFormat: "common",
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			health, err := config.
				NewHealthCheck("http://origin:8080/foo", tc.health, &config.Couper{
					Settings: config.NewDefaultSettings(),
				})
			h.Must(err)

			if tc.expectation.Request != nil && tc.expectation.Request.URL != nil {
				if *tc.expectation.Request.URL != *health.Request.URL {
					t.Errorf("Unexpected health check URI:\n\tWant: %#v\n\tGot:  %#v", tc.expectation.Request.URL, health.Request.URL)
				}
				tc.expectation.Request = nil
			}

			health.Request = nil

			if diff := cmp.Diff(tc.expectation, *health); diff != "" {
				t.Errorf("Unexpected health options:\n\n%s", diff)
			}
		})
	}
}
