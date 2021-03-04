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

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server"
)

func TestEndpoint_RoundTrip_SetGetBody_LimitBody(t *testing.T) {
	type testCase struct {
		name    string
		limit   string
		payload string
		wantErr error
	}

	for _, testcase := range []testCase{
		{"/w well sized limit", "12MiB", "content", nil},
		{"/w zero limit", "0", "01", errors.EndpointReqBodySizeExceeded},
		{"/w limit /w oversize body", "4B", "12345", errors.EndpointReqBodySizeExceeded},
	} {
		t.Run(testcase.name, func(subT *testing.T) {
			helper := test.New(subT)

			logger, _ := logrustest.NewNullLogger()
			log := logger.WithContext(context.Background())

			bodyLimit, _ := handler.ParseBodyLimit(testcase.limit)
			epHandler := handler.NewEndpoint(&handler.EndpointOptions{
				Context:      helper.NewProxyContext("set_request_headers = { x = req.post }"),
				ReqBodyLimit: bodyLimit,
			}, eval.NewENVContext(nil), log, nil, nil, nil)

			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(testcase.payload))
			err := epHandler.SetGetBody(req)
			if !reflect.DeepEqual(err, testcase.wantErr) {
				subT.Errorf("Expected '%v', got: '%v'", testcase.wantErr, err)
			}
		})
	}
}

func TestEndpoint_RoundTrip_Eval(t *testing.T) {
	type header map[string]string

	type testCase struct {
		name       string
		hcl        string
		method     string
		body       io.Reader
		wantHeader header
	}

	baseCtx := eval.NewENVContext(nil)

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
		{"GET use req.Header", `
		set_response_headers = {
			X-Method = req.method
		}`, http.MethodGet, nil, header{"X-Method": http.MethodGet}},
		{"POST use req.post", `
		set_response_headers = {
			X-Method = req.method
			X-Post = req.post.foo
		}`, http.MethodPost, strings.NewReader(`foo=bar`), header{
			"X-Method": http.MethodPost,
			"X-Post":   "bar",
		}},
	}

	evalCtx := eval.NewENVContext(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			helper := test.New(subT)
			hook.Reset()

			var remain hclBody
			err := hclsimple.Decode("test.hcl", []byte(tt.hcl), baseCtx, &remain)
			helper.Must(err)

			backend := transport.NewBackend(evalCtx,
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				&transport.Config{NoProxyFromEnv: true}, logger, nil)

			ep := handler.NewEndpoint(&handler.EndpointOptions{
				Error:        errors.DefaultJSON,
				Context:      remain.Inline,
				ReqBodyLimit: 1024,
			}, evalCtx, logger, producer.Proxies{
				&producer.Proxy{Name: "default", RoundTrip: backend},
			}, nil, nil)

			req := httptest.NewRequest(tt.method, "http://couper.io", tt.body)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			rec := httptest.NewRecorder()
			rw := server.NewRWWrapper(rec, false) // crucial for working ep due to res.Write()
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
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
	}

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
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
			x-test = req.json_body.foo
		}`, defaultMethods, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": "bar"}}},
		{"method /w body", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, []string{http.MethodTrace}, test.Header{"Content-Type": "application/json"}, `{"foo": "bar"}`, want{req: test.Header{"x-test": ""}}},
		{"method /wo body", `
		origin = "` + origin.URL + `"
		set_request_headers = {
			x-test = req.json_body.foo
		}`, append(defaultMethods, http.MethodTrace),
			test.Header{"Content-Type": "application/json"}, "", want{req: test.Header{"x-test": ""}}},
	}

	evalCtx := eval.NewENVContext(nil)

	log, _ := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(method+" "+tt.name, func(subT *testing.T) {
				helper := test.New(subT)

				backend := transport.NewBackend(evalCtx,
					helper.NewProxyContext(tt.inlineCtx),
					&transport.Config{NoProxyFromEnv: true}, logger, nil)

				ep := handler.NewEndpoint(&handler.EndpointOptions{
					Error:        errors.DefaultJSON,
					Context:      hcl.EmptyBody(),
					ReqBodyLimit: 1024,
				}, evalCtx, logger, producer.Proxies{
					&producer.Proxy{Name: "default", RoundTrip: backend},
				}, nil, nil)

				var body io.Reader
				if tt.body != "" {
					body = bytes.NewBufferString(tt.body)
				}
				req := httptest.NewRequest(method, "/", body)
				tt.header.Set(req)

				rec := httptest.NewRecorder()
				rw := server.NewRWWrapper(rec, false) // crucial for working ep due to res.Write()
				ep.ServeHTTP(rw, req)
				rec.Flush()
				//res := rec.Result()

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
func TestEndpoint_RoundTripContext_Null_Eval(t *testing.T) {
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

	evalCtx := eval.NewENVContext(nil)

	log, _ := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())

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

			backend := transport.NewBackend(evalCtx,
				test.NewRemainContext("origin", "http://"+origin.Listener.Addr().String()),
				&transport.Config{NoProxyFromEnv: true}, logger, nil)

			ep := handler.NewEndpoint(&handler.EndpointOptions{
				Error:        errors.DefaultJSON,
				Context:      helper.NewProxyContext(tc.remain),
				ReqBodyLimit: 1024,
			}, evalCtx, logger, producer.Proxies{
				&producer.Proxy{Name: "default", RoundTrip: backend},
			}, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "http://localhost/", bytes.NewReader(clientPayload))
			req.Header.Set("Content-Type", "application/json")
			*req = *req.WithContext(context.WithValue(req.Context(), request.UID, fmt.Sprintf("#%.2d: %s", i+1, tc.name)))

			rec := httptest.NewRecorder()
			rw := server.NewRWWrapper(rec, false) // crucial for working ep due to res.Write()
			ep.ServeHTTP(rw, req)
			rec.Flush()
			res := rec.Result()

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
