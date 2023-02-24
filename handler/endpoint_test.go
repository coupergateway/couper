package handler_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
	"github.com/sirupsen/logrus"
)

func TestEndpoint_RoundTrip_Eval(t *testing.T) {
	type header map[string]string

	type testCase struct {
		name       string
		hcl        string
		method     string
		body       io.Reader
		wantHeader header
	}

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

	log, hook := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())

	tests := []testCase{
		{"GET use request.Header", `
		set_response_headers = {
			X-Method = request.method
		}`, http.MethodGet, nil, header{"X-Method": http.MethodGet}},
		{"POST use request.form_body", `
		set_response_headers = {
			X-Method = request.method
			X-Form_Body = request.form_body.foo
		}`, http.MethodPost, strings.NewReader(`foo=bar`), header{
			"X-Method":    http.MethodPost,
			"X-Form_Body": "bar",
		}},
	}

	evalCtx := eval.NewDefaultContext()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			var remain hclBody
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), evalCtx.HCLContext(), &remain)
			helper.Must(err)

			backend := transport.NewBackend(
				hclbody.NewHCLSyntaxBodyWithStringAttr("origin", "http://"+origin.Listener.Addr().String()),
				&transport.Config{NoProxyFromEnv: true}, nil, logger)

			ep := handler.NewEndpoint(&handler.EndpointOptions{
				ErrorTemplate: errors.DefaultJSON,
				Context:       remain.Inline.(*hclsyntax.Body),
				ReqBodyLimit:  1024,
				Proxies: producer.Proxies{
					&producer.Proxy{Name: "default", RoundTrip: backend},
				},
				Requests: make(producer.Requests, 0),
			}, logger, nil)

			req := httptest.NewRequest(tt.method, "http://couper.io", tt.body)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			helper.Must(eval.SetGetBody(req, eval.BufferRequest, 1024))
			*req = *req.WithContext(evalCtx.WithClientRequest(req))

			rec := httptest.NewRecorder()
			rw := writer.NewResponseWriter(rec, "") // crucial for working ep due to res.Write()
			ep.ServeHTTP(rw, req)
			rec.Flush()
			res := rec.Result()

			if res == nil {
				subT.Log(hook.LastEntry().String())
				subT.Errorf("Expected a response")
				return
			}

			if res.StatusCode != http.StatusNoContent {
				subT.Errorf("Expected StatusNoContent 204, got: %q %d", res.Status, res.StatusCode)
				subT.Log(hook.LastEntry().String())
			}

			for k, v := range tt.wantHeader {
				if got := res.Header.Get(k); got != v {
					subT.Errorf("Expected value for header %q: %q, got: %q", k, v, got)
					subT.Log(hook.LastEntry().String())
				}
			}

		})
	}
}

func TestEndpoint_RoundTripContext_Variables_json_body(t *testing.T) {
	type want struct {
		req test.Header
	}

	defaultMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
	}

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// reflect req headers
		for k, v := range r.Header {
			if !strings.HasPrefix(strings.ToLower(k), "x-") {
				continue
			}
			rw.Header()[k] = v
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name      string
		inlineCtx string
		methods   []string
		header    test.Header
		body      string
		want      want
	}{
		{"method /w body", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = request.json_body.foo
		}`, defaultMethods, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": "bar"}},
		},
		{"method /w body +json content-type", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = request.json_body.foo
		}`, defaultMethods, test.Header{"Content-Type": "applicAtion/foo+jsOn"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": "bar"}},
		},
		{"method /w body wrong content-type", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = request.json_body.foo
		}`, defaultMethods, test.Header{"Content-Type": "application/fooson"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": ""}},
		},
		{"method /w body", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = request.json_body.foo
		}`, []string{http.MethodTrace, http.MethodHead}, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": ""}}},
		{"method /wo body", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = request.json_body.foo
		}`, append(defaultMethods, http.MethodTrace),
			test.Header{"Content-Type": "application/json"}, "", want{req: test.Header{"x-test": ""}}},
	}

	log, _ := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(method+" "+tt.name, func(subT *testing.T) {
				helper := test.New(subT)

				backend := transport.NewBackend(
					helper.NewInlineContext(tt.inlineCtx),
					&transport.Config{NoProxyFromEnv: true}, nil, logger)

				ep := handler.NewEndpoint(&handler.EndpointOptions{
					ErrorTemplate: errors.DefaultJSON,
					Context:       &hclsyntax.Body{},
					ReqBodyLimit:  1024,
					Proxies: producer.Proxies{
						&producer.Proxy{Name: "default", RoundTrip: backend},
					},
					Requests: make(producer.Requests, 0),
				}, logger, nil)

				var body io.Reader
				if tt.body != "" {
					body = bytes.NewBufferString(tt.body)
				}
				req := httptest.NewRequest(method, "/", body)
				tt.header.Set(req)

				// normally injected by server/http
				helper.Must(eval.SetGetBody(req, eval.BufferRequest, 1024))
				*req = *req.WithContext(eval.NewDefaultContext().WithClientRequest(req))

				rec := httptest.NewRecorder()
				rw := writer.NewResponseWriter(rec, "") // crucial for working ep due to res.Write()
				ep.ServeHTTP(rw, req)
				rec.Flush()
				res := rec.Result()

				for k, v := range tt.want.req {
					if res.Header.Get(k) != v {
						subT.Errorf("want: %q for key %q, got: %q", v, k, res.Header.Get(k))
					}
				}
			})
		}
	}
}

// TestProxy_SetRoundtripContext_Null_Eval tests the handling with non-existing references or cty.Null evaluations.
func TestEndpoint_RoundTripContext_Null_Eval(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		name       string
		remain     string
		ct         string
		expHeaders test.Header
	}

	clientPayload := []byte(`{ "client": true, "origin": false, "nil": null }`)
	originPayload := []byte(`{ "client": false, "origin": true, "nil": null }`)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientData, err := io.ReadAll(r.Body)
		helper.Must(err)
		if !bytes.Equal(clientData, clientPayload) {
			t.Errorf("Expected a request with client payload, got %q", string(clientData))
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "" {
			rw.Header().Set("Content-Type", ct)
		} else {
			rw.Header().Set("Content-Type", "application/json")
		}
		_, err = rw.Write(originPayload)
		helper.Must(err)
	}))

	log, _ := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())

	for _, tc := range []testCase{
		{"no eval", `path = "/"`, "", test.Header{}},
		{"json_body client field", `set_response_headers = { "x-client" = "my-val-x-${request.json_body.client}" }`, "",
			test.Header{
				"x-client": "my-val-x-true",
			}},
		{"json_body request/response", `set_response_headers = {
				x-client = "my-val-x-${request.json_body.client}"
				x-client2 = request.body
				x-origin = "my-val-y-${backend_responses.default.json_body.origin}"
				x-origin2 = backend_responses.default.body
			}`, "",
			test.Header{
				"x-client":  "my-val-x-true",
				"x-client2": `{ "client": true, "origin": false, "nil": null }`,
				"x-origin":  "my-val-y-true",
				"x-origin2": `{ "client": false, "origin": true, "nil": null }`,
			}},
		{"json_body request/response json variant", `set_response_headers = {
				x-client = "my-val-x-${request.json_body.client}"
				x-origin = "my-val-y-${backend_responses.default.json_body.origin}"
			}`, "application/foo+json",
			test.Header{
				"x-client": "my-val-x-true",
				"x-origin": "my-val-y-true",
			}},
		{"json_body non existing shared parent", `set_response_headers = {
				x-client = request.json_body.not-there
				x-client-nested = request.json_body.not-there.nested
			}`, "application/foo+json",
			test.Header{
				"x-client":        "",
				"x-client-nested": "",
			}},
		{"json_body non existing field", `set_response_headers = {
"${backend_responses.default.json_body.not-there}" = "my-val-0-${backend_responses.default.json_body.origin}"
"${request.json_body.client}-my-val-a" = "my-val-b-${backend_responses.default.json_body.client}"
}`, "",
			test.Header{"true-my-val-a": "my-val-b-false"}},
		{"json_body null value", `set_response_headers = { "x-null" = "${backend_responses.default.json_body.nil}" }`, "", test.Header{"x-null": ""}},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			h := test.New(subT)

			backend := transport.NewBackend(
				hclbody.NewHCLSyntaxBodyWithStringAttr("origin", "http://"+origin.Listener.Addr().String()),
				&transport.Config{NoProxyFromEnv: true}, nil, logger)

			bufOpts := eval.MustBuffer(helper.NewInlineContext(tc.remain))

			ep := handler.NewEndpoint(&handler.EndpointOptions{
				BufferOpts:    bufOpts,
				Context:       helper.NewInlineContext(tc.remain),
				ErrorTemplate: errors.DefaultJSON,
				ReqBodyLimit:  1024,
				Proxies: producer.Proxies{
					&producer.Proxy{Name: "default", RoundTrip: backend},
				},
				Requests: make(producer.Requests, 0),
			}, logger, nil)

			req := httptest.NewRequest(http.MethodPost, "http://localhost/", bytes.NewReader(clientPayload))
			helper.Must(eval.SetGetBody(req, bufOpts, 1024))
			if tc.ct != "" {
				req.Header.Set("Content-Type", tc.ct)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
			req = req.WithContext(eval.NewDefaultContext().WithClientRequest(req))

			rec := httptest.NewRecorder()
			rw := writer.NewResponseWriter(rec, "") // crucial for working ep due to res.Write()
			ep.ServeHTTP(rw, req)
			rec.Flush()
			res := rec.Result()

			if res.StatusCode != http.StatusOK {
				subT.Errorf("Expected StatusOK, got: %d", res.StatusCode)
			}

			originData, err := io.ReadAll(res.Body)
			h.Must(err)

			if !bytes.Equal(originPayload, originData) {
				subT.Errorf("Expected same origin payload, got:\n%s\nlog message:\n", string(originData))
			}

			for k, v := range tc.expHeaders {
				if res.Header.Get(k) != v {
					subT.Errorf("%q: Expected header %q value: %q, got: %q", tc.name, k, v, res.Header.Get(k))
				}
			}
		})

	}

	origin.Close()
}

var _ producer.Roundtrip = &mockProducerResult{}

type mockProducerResult struct {
	rt http.RoundTripper
}

func (m *mockProducerResult) Produce(r *http.Request, _ *sync.Map) chan *producer.Result {
	result := make(chan *producer.Result, 1)
	defer close(result)

	if m == nil || m.rt == nil {
		return result
	}

	res, err := m.rt.RoundTrip(r)
	result <- &producer.Result{
		RoundTripName: "default",
		Beresp:        res,
		Err:           err,
	}
	return result
}

func (m *mockProducerResult) Len() int {
	return 1
}

func (m *mockProducerResult) Names() []string {
	return []string{"default"}
}

func TestEndpoint_ServeHTTP_FaultyDefaultResponse(t *testing.T) {
	log, hook := test.NewLogger()

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ico, _ := os.ReadFile("testdata/file/favicon.ico")

		rw.Header().Set("Content-Encoding", "gzip")  // wrong
		rw.Header().Set("Content-Type", "text/html") // wrong
		rw.Header().Set("Cache-Control", "no-cache, no-store, max-age=0")

		_, err := rw.Write(ico)
		if err != nil {
			t.Error(err)
		}
	}))
	defer origin.Close()

	rt := transport.NewBackend(
		hclbody.NewHCLSyntaxBodyWithStringAttr("origin", origin.URL), &transport.Config{},
		&transport.BackendOptions{}, log.WithContext(context.Background()))

	mockProducer := &mockProducerResult{rt}

	ep := handler.NewEndpoint(&handler.EndpointOptions{
		Context:       &hclsyntax.Body{},
		ErrorTemplate: errors.DefaultJSON,
		Proxies:       &mockProducerResult{},
		Requests:      mockProducer,
	}, log.WithContext(context.Background()), nil)

	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "http://", nil).WithContext(ctx)
	ctx = eval.NewDefaultContext().WithClientRequest(req)
	ctx = context.WithValue(ctx, request.UID, "test123")

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	ep.ServeHTTP(rw, req.Clone(ctx))
	res := rec.Result()

	if res.StatusCode == 0 {
		t.Errorf("Fatal error: response status is zero")
		if res.Header.Get("Couper-Error") != "internal server error" {
			t.Errorf("Expected internal server error, got: %s", res.Header.Get("Couper-Error"))
		}
	} else if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status ok, got: %v", res.StatusCode)
	}

	for _, e := range hook.AllEntries() {
		if e.Level != logrus.ErrorLevel {
			continue
		}
		if e.Message != "backend error: body reset: gzip: invalid header" {
			t.Errorf("Unexpected error message: %s", e.Message)
		}
	}
}

func TestEndpoint_ServeHTTP_Cancel(t *testing.T) {
	log, hook := test.NewLogger()
	slowOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 5)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer slowOrigin.Close()

	ctx, cancelFn := context.WithCancel(context.WithValue(context.Background(), request.UID, "test123"))
	ctx = context.WithValue(ctx, request.StartTime, time.Now())

	rt := transport.NewBackend(
		hclbody.NewHCLSyntaxBodyWithStringAttr("origin", slowOrigin.URL), &transport.Config{},
		&transport.BackendOptions{}, log.WithContext(context.Background()))

	mockProducer := &mockProducerResult{rt}

	ep := handler.NewEndpoint(&handler.EndpointOptions{
		Context:       &hclsyntax.Body{},
		ErrorTemplate: errors.DefaultJSON,
		Proxies:       &mockProducerResult{},
		Requests:      mockProducer,
	}, log.WithContext(ctx), nil)

	req := httptest.NewRequest(http.MethodGet, "https://couper.io/", nil)
	ctx = eval.NewDefaultContext().WithClientRequest(req.WithContext(ctx))

	start := time.Now()
	go func() {
		time.Sleep(time.Second)
		cancelFn()
	}()

	rec := httptest.NewRecorder()
	access := logging.NewAccessLog(&logging.Config{}, log)

	outreq := req.WithContext(ctx)
	ep.ServeHTTP(rec, outreq)
	access.Do(rec, outreq)
	rec.Flush()

	elapsed := time.Since(start)
	if elapsed > time.Second+(time.Millisecond*50) {
		t.Error("Expected canceled request")
	}

	for _, e := range hook.AllEntries() {
		if e.Message == "client request error: context canceled" {
			return
		}
	}

	t.Error("Expected context canceled access log, got:\n")
	for _, e := range hook.AllEntries() {
		println(e.String())
	}
}
