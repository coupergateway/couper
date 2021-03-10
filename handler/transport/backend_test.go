package transport_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
)

func TestBackend_RoundTrip_Timings(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb and overall timeout
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name        string
		context     hcl.Body
		tconf       *transport.Config
		req         *http.Request
		expectedErr string
	}{
		{"with zero timings", test.NewRemainContext("origin", origin.URL), &transport.Config{}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), ""},
		{"with overall timeout", test.NewRemainContext("origin", origin.URL), &transport.Config{Timeout: time.Second / 2, ConnectTimeout: time.Minute}, httptest.NewRequest(http.MethodHead, "http://1.2.3.5/", nil), "context deadline exceeded"},
		{"with connect timeout", test.NewRemainContext("origin", "http://blackhole.webpagetest.org"), &transport.Config{ConnectTimeout: time.Second / 2}, httptest.NewRequest(http.MethodGet, "http://1.2.3.6/", nil), "i/o timeout"},
		{"with ttfb timeout", test.NewRemainContext("origin", origin.URL), &transport.Config{TTFBTimeout: time.Second}, httptest.NewRequest(http.MethodHead, "http://1.2.3.7/", nil), "timeout awaiting response headers"},
	}

	logger, hook := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			tt.tconf.NoProxyFromEnv = true // use origin addr from transport.Config
			backend := transport.NewBackend(tt.context, tt.tconf, log, nil)

			_, err := backend.RoundTrip(tt.req)
			if err != nil && tt.expectedErr == "" {
				t.Error(err)
				return
			}

			if tt.expectedErr != "" &&
				(err == nil || !strings.HasSuffix(err.Error(), tt.expectedErr)) {
				t.Errorf("Expected err %s, got: %v", tt.expectedErr, err)
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

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})
	backend := transport.NewBackend(hclBody, &transport.Config{}, log, nil)

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

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})

	backend := transport.NewBackend(hclBody, &transport.Config{
		Origin: origin.URL,
	}, log, nil)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if l := res.Header.Get("Content-Length"); l != "60" {
		t.Errorf("Unexpected C/L: %s", l)
	}

	n, err := io.Copy(ioutil.Discard, res.Body)
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

	openAPIYAML := helper.NewOpenAPIConf("/get")

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
			"/get",
			"",
			"",
		},
		{
			"invalid request",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodPost,
			"/get",
			"Upstream request validation failed",
			"request validation: 'POST /get': Path doesn't support the HTTP method",
		},
		{
			"invalid request, IgnoreRequestViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreRequestViolations: true, IgnoreResponseViolations: true},
			http.MethodPost,
			"/get",
			"",
			"request validation: 'POST /get': Path doesn't support the HTTP method",
		},
		{
			"invalid response",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/get?404",
			"Upstream response validation failed",
			"response validation: status is not supported",
		},
		{
			"invalid response, IgnoreResponseViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreResponseViolations: true},
			http.MethodGet,
			"/get?404",
			"",
			"response validation: status is not supported",
		},
	}

	logger, hook := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			openapiValidatorOptions, err := validation.NewOpenAPIOptionsFromBytes(tt.openapi, openAPIYAML)
			if err != nil {
				subT.Fatal(err)
			}
			content := helper.NewProxyContext(`
				origin = "` + origin.URL + `"
			`)

			backend := transport.NewBackend(content, &transport.Config{}, log, openapiValidatorOptions)

			req := httptest.NewRequest(tt.requestMethod, "http://1.2.3.4"+tt.requestPath, nil)

			_, err = backend.RoundTrip(req)
			if err != nil && tt.expectedErr == "" {
				subT.Error(err)
				return
			}

			if tt.expectedErr != "" && (err == nil || err.Error() != tt.expectedErr) {
				subT.Errorf("Expected error %s, got: %v", tt.expectedErr, err)
				subT.Log(hook.LastEntry().Message)
			}

			entry := hook.LastEntry()
			if tt.expectedLogMessage != "" {
				if data, ok := entry.Data["validation"]; ok {
					for _, err := range data.([]string) {
						if err == tt.expectedLogMessage {
							return
						}
					}
					for _, err := range data.([]string) {
						subT.Log(err)
					}
				}
				subT.Errorf("expected matching validation error logs:\n\t%s\n\tgot: nothing", tt.expectedLogMessage)
			}

		})
	}
}

func TestBackend_director(t *testing.T) {
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

			backend := transport.NewBackend(hclContext, &transport.Config{
				Timeout: time.Second,
			}, nullLog, nil)

			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			*req = *req.Clone(tt.ctx)

			_, _ = backend.RoundTrip(req) // implicit director()

			attr, _ := hclContext.JustAttributes()
			hostnameExp, ok := attr["hostname"]

			if !ok && tt.expReq.Host != req.Host {
				t.Errorf("expected same host value, want: %q, got: %q", req.Host, tt.expReq.Host)
			} else if ok {
				hostVal, _ := hostnameExp.Expr.Value(eval.NewContext(nil).HCLContext())
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

// TestProxy_BufferingOptions tests the option interaction with enabled/disabled validation and
// the requirement for buffering to read the post or json body.
func TestProxy_BufferingOptions(t *testing.T) {
	t.Skip("TODO: variable buffering option configurable again")
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

	log, _ := logrustest.NewNullLogger()
	nullLog := log.WithContext(nil)

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

			backend := transport.NewBackend(configload.MergeBodies([]hcl.Body{
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				helper.NewProxyContext(tc.remain),
			}), &transport.Config{}, nullLog, newOptions())

			upstreamLog := backend.(*logging.UpstreamLog)
			backendHandler := reflect.ValueOf(upstreamLog).Elem().FieldByName("next")              // private field: ro
			configuredOption := reflect.ValueOf(backendHandler).Elem().FieldByName("bufferOption") // private field: ro
			var opt eval.BufferOption
			if configuredOption.IsValid() {
				opt = eval.BufferOption(configuredOption.Uint())
			} else {
				st.Errorf("Field read out failed: bufferOption")
			}
			if (opt & tc.expectedOption) != tc.expectedOption {
				st.Errorf("Expected option: %#v, got: %#v", tc.expectedOption, opt)
			}

			req := httptest.NewRequest(http.MethodGet, "http://localhost/", bytes.NewReader(clientPayload))
			req.Header.Set("Content-Type", "application/json")
			*req = *req.WithContext(context.WithValue(req.Context(), request.UID, fmt.Sprintf("#%.2d: %s", i+1, tc.name)))

			res, err := backend.RoundTrip(req)
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
