package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
)

const testdataPath = "testdata/endpoints"

func TestEndpoints_Protected404(t *testing.T) {
	client := newClient()

	type expectation struct {
		ResponseStatus int
	}

	type testCase struct {
		auth string
		path string
		exp  expectation
	}

	shutdown, _ := newCouper("testdata/endpoints/07_couper.hcl", test.New(t))
	defer shutdown()

	for _, tc := range []testCase{
		{"", "/v1/anything", expectation{}},
		{"secret", "/v1/anything", expectation{http.StatusOK}},
		{"", "/v1/xxx", expectation{}},
		{"secret", "/v1/xxx", expectation{http.StatusNotFound}},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			if tc.auth != "" {
				req.SetBasicAuth("", tc.auth)
			}

			res, err := client.Do(req)
			helper.Must(err)

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				subT.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				subT.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
			}
		})
	}
}

func TestEndpoints_ProxyReqRes(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "01_couper.hcl"), helper)
	defer shutdown()

	defer func() {
		if !t.Failed() {
			return
		}
		for _, e := range logHook.AllEntries() {
			println(e.String())
		}
	}()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 5 {
		t.Fatalf("Expected 5 log entries, given %d", l)
	}

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "1616" {
		t.Errorf("Expected body 1616, given %s", resBytes)
	}
}

func TestEndpoints_BerespBody(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "08_couper.hcl"), helper)
	defer shutdown()

	defer func() {
		if !t.Failed() {
			return
		}
		for _, e := range logHook.AllEntries() {
			println(e.String())
		}
	}()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/pdf", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if !bytes.HasPrefix(resBytes, []byte("%PDF-1.6")) {
		t.Errorf("Expected PDF file, given %s", resBytes)
	}

	if val := res.Header.Get("x-body"); !strings.HasPrefix(val, "%PDF-1.6") {
		t.Errorf("Expected PDF file content, got: %q", val)
	}
}

func TestEndpoints_ReqBody(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "08_couper.hcl"), helper)
	defer shutdown()

	defer func() {
		if !t.Failed() {
			return
		}
		for _, e := range logHook.AllEntries() {
			println(e.String())
		}
	}()

	payload := "content"
	req, err := http.NewRequest(http.MethodPost, "http://example.com:8080/post", bytes.NewBufferString(payload))
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	type result struct {
		Body    string
		Headers http.Header
	}

	r := &result{}
	helper.Must(json.Unmarshal(resBytes, r))

	if r.Body != payload {
		t.Errorf("Want: content, got: %v", r.Body)
	}

	if r.Headers.Get("Content-Length") != strconv.Itoa(len(payload)) {
		t.Errorf("Expected content-length: %d", len(payload))
	}
}

func TestEndpoints_ProxyReqResCancel(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "01_couper.hcl"), helper)
	defer shutdown()

	defer func() {
		if t.Failed() {
			for _, e := range logHook.Entries {
				println(e.String())
				if d, ok := e.Data["panic"]; ok {
					println(d)
				}
			}
		}
	}()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1?delay=5s", nil)
	helper.Must(err)

	ctx, cancel := context.WithCancel(req.Context())
	*req = *req.WithContext(ctx)

	logHook.Reset()

	go func() {
		time.Sleep(time.Second / 2)
		cancel()
	}()

	_, err = client.Do(req)
	if err == nil {
		t.Error("expected a cancel error")
	} else {
		if cancelErr := errors.Unwrap(err); cancelErr != context.Canceled {
			t.Error("expected a cancel error")
		}
	}
}

func TestEndpoints_RequestLimit(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, _ := newCouper(path.Join(testdataPath, "06_couper.hcl"), helper)
	defer shutdown()

	body := strings.NewReader(`{"foo" = "bar"}`)

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", body)
	helper.Must(err)

	req.SetBasicAuth("", "qwertz")

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, given %d", res.StatusCode)
	}
}

func TestEndpoints_Res(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "02_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 1 {
		t.Fatalf("Expected 1 log entries, given %d", l)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "string" {
		t.Errorf("Expected body 'string', given %s", resBytes)
	}
}

func TestEndpoints_UpstreamBasicAuthAndXFF(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, _ := newCouper(path.Join(testdataPath, "03_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/anything", nil)
	helper.Must(err)

	req.Header.Set("X-User", "user")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	type expectation struct {
		Headers http.Header
	}

	var jsonResult expectation
	err = json.Unmarshal(resBytes, &jsonResult)
	helper.Must(err)

	// The "dXNlcjpwYXNz" is base64encode("user:pass") from the HCL file.
	if v := jsonResult.Headers.Get("Authorization"); v != "Basic dXNlcjpwYXNz" {
		t.Errorf("Expected a valid 'Authorization' header, given '%s'", v)
	}

	if v := jsonResult.Headers.Get("X-Forwarded-For"); v != "1.2.3.4" {
		t.Errorf("Unexpected XFF header given '%s'", v)
	}
}

func TestEndpoints_Muxing(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, _ := newCouper(path.Join(testdataPath, "05_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "s1" {
		t.Errorf("Expected body 's1', given %s", resBytes)
	}
}

func TestEndpoints_DoNotExecuteResponseOnErrors(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, _ := newCouper(path.Join(testdataPath, "09_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if !bytes.Contains(resBytes, []byte("<html>configuration error</html>")) {
		t.Errorf("Expected body '<html>configuration error</html>', given '%s'", resBytes)
	}

	// header from error handling is set
	if v := res.Header.Get("couper-error"); v != "configuration error" {
		t.Errorf("want couper-error 'configuration error', got %q", v)
	}

	// happy path headers not set
	if res.Header.Get("x-backend") != "" {
		t.Errorf("backend.set_response_headers should not have been run")
	}

	if res.Header.Get("x-endpoint") != "" {
		t.Errorf("endpoint.set_response_headers should not have been run")
	}
}

func TestHTTPServer_NoGzipForSmallContent(t *testing.T) {
	client := newClient()

	confPath := path.Join("testdata/endpoints/10_couper.hcl")
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	type testCase struct {
		path   string
		expLen string
		expCE  string
	}

	for _, tc := range []testCase{
		{"/0", "0", ""},
		{"/10", "10", ""},
		{"/59", "59", ""},
		{"/60", "47", "gzip"},
		{"/x", "1731", "gzip"},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.org:9898"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Accept-Encoding", "gzip")

			res, err := client.Do(req)
			helper.Must(err)

			if val := res.Header.Get("Content-Encoding"); val != tc.expCE {
				subT.Errorf("%s: Expected Content-Encoding '%s', got: '%s'", tc.path, tc.expCE, val)
			}
			if val := res.Header.Get("Content-Length"); val != tc.expLen {
				subT.Errorf("%s: Expected Content-Length '%s', got: '%s'", tc.path, tc.expLen, val)
			}
		})
	}
}

func TestEndpointSequence(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(path.Join(testdataPath, "11_couper.hcl"), helper)
	defer shutdown()

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Y-Value", "my-value")
		if req.Header.Get("Accept") == "application/json" {
			rw.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(rw, `{"value":"%s"}`, req.Header.Get("X-Value"))
		} else {
			rw.WriteHeader(http.StatusNoContent)
		}
	}))
	defer origin.Close()

	type log map[string]string

	type testcase struct {
		name           string
		path           string
		expectedHeader test.Header
		expectedBody   string
		expectedLog    log
	}

	for _, tc := range []testcase{
		{"simple request sequence", "/simple", test.Header{"x": "my-value"}, `{"value":"my-value"}`, log{"default": "resolve"}},
		{"simple request/proxy sequence", "/simple-proxy", test.Header{"x": "my-value", "y": `{"value":"my-proxy-value"}`}, "", log{"default": "resolve"}},
		{"simple proxy/request sequence", "/simple-proxy-named", test.Header{"x": "my-value"}, "", log{"default": "resolve"}},
		{"complex request/proxy sequence", "/complex-proxy", test.Header{"x": "my-value"}, "", log{"default": "resolve", "resolve": "resolve_first"}},
		{"complex request/proxy sequences", "/parallel-complex-proxy", test.Header{"x": "my-value", "y": "my-value", "z": "my-value"}, "", log{"default": "resolve", "resolve": "resolve_first"}},
		{"complex nested request/proxy sequences", "/parallel-complex-nested", test.Header{
			"a": "my-value",
			"b": "my-value",
			"x": "my-value",
			"y": "my-value",
		}, "", log{
			"default":       "resolve",
			"resolve":       "resolve_first",
			"resolve_gamma": "resolve_gamma_first",
			"last":          "resolve,resolve_gamma",
		}},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()

			h := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080"+tc.path, nil)
			h.Must(err)

			req.Header.Set("Origin", origin.URL)

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusOK {
				st.Fatal("expected status ok")
			}

			for k, v := range tc.expectedHeader {
				if hv := res.Header.Get(k); hv != v {
					st.Errorf("%q: want %q, got %q", k, v, hv)
					break
				}
			}

			if tc.expectedBody != "" {
				result, err := io.ReadAll(res.Body)
				h.Must(err)

				if tc.expectedBody != string(result) {
					st.Errorf("unexpected body:\n%s", cmp.Diff(tc.expectedBody, string(result)))
				}
			}

			for _, e := range hook.AllEntries() {
				if e.Data["type"] != "couper_backend" {
					continue
				}

				requestName, _ := e.Data["request"].(logging.Fields)["name"].(string)

				// test result for expected named ones
				if _, exist := tc.expectedLog[requestName]; !exist {
					continue
				}

				dependsOn, ok := e.Data["depends_on"]
				if !ok {
					st.Fatal("Expected 'depends_on' log field")
				}

				if dependsOn != tc.expectedLog[requestName] {
					st.Errorf("Expected 'depends_on' log for %q with field value: %q, got: %q", requestName, tc.expectedLog[requestName], dependsOn)
				}
			}

		})
	}

}

func TestEndpointSequenceClientCancel(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(path.Join(testdataPath, "12_couper.hcl"), helper)
	defer shutdown()

	ctx, cancel := context.WithCancel(context.Background())

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cancel()
		time.Sleep(time.Second / 2)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080/", nil)
	helper.Must(err)

	req.Header.Set("Origin", origin.URL)

	_, err = client.Do(req.WithContext(ctx))
	if err != nil && errors.Unwrap(err) != context.Canceled {
		helper.Must(err)
	}

	time.Sleep(time.Second / 2)

	logs := hook.AllEntries()

	var ctxCanceledSeen, statusOKseen bool
	for _, entry := range logs {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		path, _ := entry.Data["request"].(logging.Fields)["path"]

		if strings.Contains(entry.Message, context.Canceled.Error()) {
			ctxCanceledSeen = true
			if path != "/" {
				t.Errorf("expected '/' to fail")
			}
		}

		if entry.Message == "" && entry.Data["status"] == 200 {
			statusOKseen = true
			if path != "/reflect" {
				t.Errorf("expected '/reflect' to be ok")
			}
		}
	}

	if !ctxCanceledSeen || !statusOKseen {
		t.Errorf("Expected one sucessful and one failed backend request")
	}

}

func TestEndpointSequenceBackendTimeout(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(path.Join(testdataPath, "13_couper.hcl"), helper)
	defer shutdown()

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Second)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080/", nil)
	helper.Must(err)

	req.Header.Set("Origin", origin.URL)

	res, err := client.Do(req)
	if err != nil {
		helper.Must(err)
	}

	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("Expected status 504, got: %d", res.StatusCode)
	}

	time.Sleep(time.Second / 4)

	logs := hook.AllEntries()

	var ctxDeadlineSeen, statusOKseen bool
	for _, entry := range logs {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		path, _ := entry.Data["request"].(logging.Fields)["path"]

		if entry.Message == "backend timeout error: deadline exceeded" {
			ctxDeadlineSeen = true
			if path != "/" {
				t.Errorf("expected '/' to fail")
			}
		}

		if entry.Message == "" && entry.Data["status"] == 200 {
			statusOKseen = true
			if path != "/reflect" {
				t.Errorf("expected '/reflect' to be ok")
			}
		}
	}

	if !ctxDeadlineSeen || !statusOKseen {
		t.Errorf("Expected one sucessful and one failed backend request")
	}

}

func TestEndpointErrorHandler(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(path.Join(testdataPath, "14_couper.hcl"), helper)
	defer shutdown()
	defer func() {
		for _, e := range hook.AllEntries() {
			t.Logf("%#v", e.Data)
		}
	}()

	type testcase struct {
		name              string
		path              string
		expectedHeader    test.Header
		expectedStatus    int
		expectedErrorType string
	}

	for _, tc := range []testcase{
		{"error_handler not triggered", "/ok", test.Header{"x": "application/json"}, http.StatusOK, ""},
		{"error_handler triggered with beresp body", "/not-ok", test.Header{"x": "200", "y": "item1"}, http.StatusTeapot, "unexpected_status"},
		{"error_handler triggered with beresp body - sequence", "/not-ok-sequence", test.Header{"x": "application/json"}, http.StatusTeapot, "unexpected_status"},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()
			h := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080"+tc.path, nil)
			h.Must(err)

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.expectedStatus {
				st.Fatalf("want: %d, got: %d", tc.expectedStatus, res.StatusCode)
			}

			for k, v := range tc.expectedHeader {
				if hv := res.Header.Get(k); hv != v {
					st.Errorf("%q: want %q, got %q", k, v, hv)
					break
				}
			}
			if tc.expectedErrorType != "" {
				for _, e := range hook.AllEntries() {
					if e.Data["type"] != "couper_access" {
						continue
					}
					if e.Data["error_type"] != tc.expectedErrorType {
						st.Errorf("want: %q, got: %q", tc.expectedErrorType, e.Data["error_type"])
					}
				}
			}
		})
	}
}
