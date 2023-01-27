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
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
)

const testdataPath = "testdata/endpoints"

func TestBackend_BackendVariable_RequestResponse(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/backends/02_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/request", nil)
	helper.Must(err)

	hook.Reset()
	res, err := client.Do(req)
	helper.Must(err)

	if res.Header.Get("X-From-Request-Header") != "bar" ||
		res.Header.Get("X-From-Request-Json-Body") != "1" ||
		res.Header.Get("X-From-Requests-Header") != "bar" ||
		res.Header.Get("X-From-Requests-Json-Body") != "1" ||
		res.Header.Get("X-From-Response-Json-Body") != "/anything" ||
		res.Header.Get("X-From-Responses-Json-Body") != "/anything" ||
		res.Header.Get("X-From-Response-Header") != "application/json" ||
		res.Header.Get("X-From-Responses-Header") != "application/json" {
		t.Errorf("Unexpected header given: %#v", res.Header)
	}

	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		responseLogs, _ := entry.Data["response"].(logging.Fields)
		data, _ := entry.Data["custom"].(logrus.Fields)

		if data != nil && entry.Data["backend"] == "anonymous_76_16" {
			expected := logrus.Fields{
				"x-from-request-body":       "grant_type=client_credentials",
				"x-from-request-form-body":  "client_credentials",
				"x-from-request-header":     "Basic cXBlYjpiZW4=",
				"x-from-response-header":    "60s",
				"x-from-response-body":      `{"access_token":"the_access_token","expires_in":60}`,
				"x-from-response-json-body": "the_access_token",
			}
			expectedHeaders := map[string]string{
				"content-type": "application/json",
				"location":     "Basic cXBlYjpiZW4=|client_credentials|60s|the_access_token",
			}

			if diff := cmp.Diff(data, expected); diff != "" {
				t.Error(diff)
			}

			if diff := cmp.Diff(responseLogs["headers"], expectedHeaders); diff != "" {
				t.Error(diff)
			}
		} else {
			expected := logrus.Fields{
				"x-from-request-json-body":   float64(1),
				"x-from-request-header":      "bar",
				"x-from-requests-json-body":  float64(1),
				"x-from-requests-header":     "bar",
				"x-from-response-header":     "application/json",
				"x-from-response-json-body":  "/anything",
				"x-from-responses-header":    "application/json",
				"x-from-responses-json-body": "/anything",
			}
			if diff := cmp.Diff(data, expected); diff != "" {
				t.Error(diff)
			}
		}
	}
}

func TestBackend_BackendVariable(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/backends/02_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	req.Header.Set("Cookie", "Cookie")
	req.Header.Set("User-Agent", "Couper")

	hook.Reset()
	res, err := client.Do(req)
	helper.Must(err)

	var check int

	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		name := entry.Data["request"].(logging.Fields)["name"]
		data, exist := entry.Data["custom"].(logrus.Fields)
		if !exist {
			t.Error("missing custom log field")
			continue
		}
		// The Cookie request header is not proxied, so *-req is not set in log.

		if name == "default" {
			check++

			if len(data) != 2 || data["default-res"] != "application/json" || data["default-ua"] != "Couper" {
				t.Errorf("unexpected data given: %#v", data)
			}
		} else if name == "request" {
			check++

			if len(data) != 2 || data["request-res"] != "text/plain; charset=utf-8" || data["request-ua"] != "" {
				t.Errorf("unexpected data given: %#v", data)
			}
		} else if name == "r1" {
			check++

			if len(data) != 2 || data["definitions-res"] != "text/plain; charset=utf-8" || data["definitions-ua"] != "" {
				t.Errorf("unexpected data given: %#v", data)
			}
		} else if name == "r2" {
			check++

			if len(data) != 2 || data["definitions-res"] != "application/json" || data["definitions-ua"] != "" {
				t.Errorf("unexpected data given: %#v", data)
			}
		}
	}

	if check != 4 {
		t.Error("missing 4 backend logs")
	}

	if got := res.Header.Get("Test-Header"); got != "application/json" {
		t.Errorf("Unexpected header given: %#v", got)
	}
}

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

	shutdown, logHook := newCouper(filepath.Join(testdataPath, "01_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 5 {
		t.Fatalf("Expected 5 log entries, given %d", l)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	if string(resBytes) != "800" {
		t.Errorf("Expected body bytes: 800, given %s", resBytes)
	}
}

func TestEndpoints_BerespBody(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(filepath.Join(testdataPath, "08_couper.hcl"), helper)
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

	shutdown, logHook := newCouper(filepath.Join(testdataPath, "08_couper.hcl"), helper)
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

	shutdown, logHook := newCouper(filepath.Join(testdataPath, "01_couper.hcl"), helper)
	defer shutdown()

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

	shutdown, _ := newCouper(filepath.Join(testdataPath, "06_couper.hcl"), helper)
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

	shutdown, logHook := newCouper(filepath.Join(testdataPath, "02_couper.hcl"), helper)
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

	shutdown, _ := newCouper(filepath.Join(testdataPath, "03_couper.hcl"), helper)
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

	shutdown, _ := newCouper(filepath.Join(testdataPath, "05_couper.hcl"), helper)
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

	shutdown, _ := newCouper(filepath.Join(testdataPath, "09_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if !bytes.Contains(resBytes, []byte("<html>backend error</html>")) {
		t.Errorf("Expected body '<html>backend error</html>', given '%s'", resBytes)
	}

	// header from error handling is set
	if v := res.Header.Get("couper-error"); v != "backend error" {
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

	confPath := filepath.Join("testdata/endpoints/10_couper.hcl")
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

func TestEndpoint_Sequence(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "11_couper.hcl"), helper)
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
		expectedDep    log
		expNumBEReq    int
	}

	for _, tc := range []testcase{
		{"simple request sequence", "/simple", test.Header{"x": "my-value"}, `{"value":"my-value"}`, log{"default": "resolve"}, 2},
		{"simple request/proxy sequence", "/simple-proxy", test.Header{"x": "my-value", "y": `{"value":"my-proxy-value"}`}, "", log{"default": "resolve"}, 2},
		{"simple proxy/request sequence", "/simple-proxy-named", test.Header{"x": "my-value"}, "", log{"default": "resolve"}, 2},
		{"complex request/proxy sequence", "/complex-proxy", test.Header{"x": "my-value"}, "", log{"default": "resolve", "resolve": "resolve_first"}, 3},
		{"complex request/proxy sequences", "/parallel-complex-proxy", test.Header{"x": "my-value", "y": "my-value", "z": "my-value"}, "", log{"default": "resolve", "resolve": "resolve_first"}, 6},
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
		}, 6},
		{"multiple request uses", "/multiple-request-uses", test.Header{}, "", log{"r2": "r1", "default": "r2,r1"}, 3},
		{"multiple proxy uses", "/multiple-proxy-uses", test.Header{}, "", log{"r2": "p1", "default": "r2,p1"}, 3},
		{"multiple sequence uses", "/multiple-sequence-uses", test.Header{}, "", log{"r2": "r1", "r4": "r2", "r3": "r2", "default": "r4,r3"}, 5},
		{"multiple parallel uses", "/multiple-parallel-uses", test.Header{}, "", log{"r4": "r2,r1", "r3": "r2,r1", "default": "r4,r3"}, 5},
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

			nbr := 0
			entries := hook.AllEntries()
			for _, e := range entries {
				if e.Data["type"] != "couper_backend" {
					continue
				}
				nbr++

				requestName, _ := e.Data["request"].(logging.Fields)["name"].(string)

				// test result for expected named ones
				if _, exist := tc.expectedDep[requestName]; !exist {
					continue
				}

				dependsOn, ok := e.Data["depends_on"]
				if !ok {
					st.Fatal("Expected 'depends_on' log field")
				}

				if dependsOn != tc.expectedDep[requestName] {
					st.Errorf("Expected 'depends_on' log for %q with field value: %q, got: %q", requestName, tc.expectedDep[requestName], dependsOn)
				}
			}
			if nbr != tc.expNumBEReq {
				st.Errorf("Expected number of backend requests: %d, got: %d", tc.expNumBEReq, nbr)
			}

			if !st.Failed() {
				return
			}
			for _, e := range entries {
				st.Logf("%#v", e.Data)
			}
		})
	}

}

func TestEndpoint_Sequence_ClientCancel(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "12_couper.hcl"), helper)
	defer shutdown()

	ctx, cancel := context.WithCancel(context.Background())

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cancel()
		time.Sleep(time.Second / 2)
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080/", nil)
	helper.Must(err)

	req.Header.Set("Origin", origin.URL)

	_, err = client.Do(req.WithContext(ctx))
	if err != nil && errors.Unwrap(err) != context.Canceled {
		helper.Must(err)
	}

	time.Sleep(time.Second * 2)

	logs := hook.AllEntries()

	if len(logs) == 0 {
		t.Fatal("missing logs")
	}

	var ctxCanceledSeen, statusOKseen bool
	for _, entry := range logs {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		path := entry.Data["request"].(logging.Fields)["path"]

		switch path {
		case "/":
			isCancelErr := strings.Contains(entry.Message, context.Canceled.Error())
			hasStatusZero := entry.Data["status"] == 0
			ctxCanceledSeen = isCancelErr && hasStatusZero && entry.Level == logrus.ErrorLevel
		case "/reflect":
			request := entry.Data["request"].(logging.Fields)
			if request["name"] != "resolve_first" {
				continue
			}
			isCancelErr := strings.Contains(entry.Message, context.Canceled.Error())
			hasStatusOK := entry.Data["status"] == http.StatusOK
			statusOKseen = !isCancelErr && hasStatusOK && entry.Level == logrus.InfoLevel
		}
	}

	if !ctxCanceledSeen || !statusOKseen {
		t.Errorf("Expected one successful and one failed backend request")
	}
}

func TestEndpoint_Sequence_BackendTimeout(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "13_couper.hcl"), helper)
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

	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("Expected status 502, got: %d", res.StatusCode)
	}

	time.Sleep(time.Second / 4)

	logs := hook.AllEntries()

	var ctxDeadlineSeen, statusOKseen bool
	for _, entry := range logs {
		if entry.Data["type"] != "couper_backend" {
			continue
		}

		path := entry.Data["request"].(logging.Fields)["path"]

		switch path {
		case "/":
			isDeadlineErr := entry.Message == "backend error: anonymous_3_23: deadline exceeded"
			hasStatusZero := entry.Data["status"] == 0
			ctxDeadlineSeen = isDeadlineErr && hasStatusZero && entry.Level == logrus.ErrorLevel
		case "/reflect":
			hasStatusOK := entry.Data["status"] == http.StatusOK
			statusOKseen = hasStatusOK && entry.Level == logrus.InfoLevel
		}
	}

	if !ctxDeadlineSeen || !statusOKseen {
		t.Errorf("Expected one successful and one failed backend request")
	}
}

func TestEndpoint_Sequence_NestedDefaultRequest(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, _ := newCouper(filepath.Join(testdataPath, "19_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusOK, got: %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	exp := `{"data":[{"features":1},{"features":2}]}`
	if !bytes.Equal([]byte(exp), b) {
		t.Errorf("expected %v, got %v", exp, string(b))
	}
}

func TestEndpointCyclicSequence(t *testing.T) {
	for _, testcase := range []struct{ file, exp string }{
		{file: "15_couper.hcl", exp: "circular sequence reference: a,b,a"},
		{file: "16_couper.hcl", exp: "circular sequence reference: a,aa,aaa,a"},
		{file: "20_couper.hcl", exp: ""},
	} {
		t.Run(testcase.file, func(st *testing.T) {
			// since we will switch the working dir, reset afterwards
			defer cleanup(func() {}, test.New(t))

			path := filepath.Join(testdataPath, testcase.file)
			_, err := configload.LoadFile(path, "")

			diags, ok := err.(*hcl.Diagnostic)
			if !ok && testcase.exp != "" {
				st.Errorf("Expected a cyclic hcl diagnostics error, got: %v", reflect.TypeOf(err))
				st.Fatal(err, path)
			} else if ok && testcase.exp == "" {
				st.Errorf("Expected no cyclic hcl diagnostics error, got: %v", reflect.TypeOf(err))
				st.Fatal(err, path)
			}

			if testcase.exp != "" && diags.Detail != testcase.exp {
				st.Errorf("\nWant:\t%s\nGot:\t%s", testcase.exp, diags.Detail)
			}
		})
	}
}

func TestEndpointErrorHandler(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "14_couper.hcl"), helper)
	defer shutdown()
	defer func() {
		if !t.Failed() {
			return
		}
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
		{"error_handler triggered with beresp body handled by endpoint", "/not-ok-endpoint", test.Header{"x": "200", "y": "item1"}, http.StatusTeapot, "unexpected_status"},
		{"error_handler triggered with beresp body - sequence", "/not-ok-sequence", test.Header{"x": "application/json"}, http.StatusTeapot, "unexpected_status"},

		{"unexpected status; handlers for unexpected_status, sequence, endpoint", "/1.1", test.Header{"handled-by": "unexpected_status"}, http.StatusOK, "unexpected_status"},
		{"unexpected status; handlers for sequence, endpoint", "/1.2", test.Header{"handled-by": "endpoint"}, http.StatusOK, "unexpected_status"},
		{"unexpected status; handler for sequence", "/1.3", test.Header{"handled-by": "sequence"}, http.StatusOK, "unexpected_status"},
		{"unexpected status; handler for endpoint", "/1.4", test.Header{"handled-by": "endpoint"}, http.StatusOK, "unexpected_status"},
		{"unexpected status; no handlers", "/1.5", test.Header{"couper-error": "endpoint error"}, http.StatusBadGateway, "unexpected_status"},

		{"backend timeout; handlers for backend_timeout, backend, sequence, endpoint", "/2.1", test.Header{"handled-by": "backend_timeout"}, http.StatusOK, "backend_timeout"},
		{"backend timeout; handlers for backend, sequence, endpoint", "/2.2", test.Header{"handled-by": "backend"}, http.StatusOK, "backend_timeout"},
		{"backend timeout; handlers for sequence, endpoint", "/2.3", test.Header{"handled-by": "sequence"}, http.StatusOK, "backend_timeout"},
		{"backend timeout; handler for endpoint", "/2.4", test.Header{"handled-by": "endpoint"}, http.StatusOK, "backend_timeout"},
		{"backend timeout; no handler", "/2.5", test.Header{"couper-error": "endpoint error"}, http.StatusBadGateway, "backend_timeout"},

		{"backend openapi validation; handlers for backend_openapi_validation, backend, sequence, endpoint", "/3.1", test.Header{"handled-by": "backend_openapi_validation"}, http.StatusOK, "backend_openapi_validation"},
		{"backend openapi validation; handlers for backend, sequence, endpoint", "/3.2", test.Header{"handled-by": "backend"}, http.StatusOK, "backend_openapi_validation"},
		{"backend openapi validation; handlers for sequence, endpoint", "/3.3", test.Header{"handled-by": "sequence"}, http.StatusOK, "backend_openapi_validation"},
		{"backend openapi validation; handler for endpoint", "/3.4", test.Header{"handled-by": "endpoint"}, http.StatusOK, "backend_openapi_validation"},
		{"backend openapi validation; no handler", "/3.5", test.Header{"couper-error": "endpoint error"}, http.StatusBadGateway, "backend_openapi_validation"},
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

func TestEndpointSequenceBreak(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "14_couper.hcl"), helper)
	defer shutdown()
	defer func() {
		if !t.Failed() {
			return
		}
		for _, e := range hook.AllEntries() {
			t.Logf("%#v", e.Data)
		}
	}()

	type testcase struct {
		name              string
		path              string
		expectedErrorType string
		expBERNames       []string
	}

	for _, tc := range []testcase{
		{"sequence break unexpected_status", "/sequence-break-unexpected_status", "unexpected_status", []string{"resolve"}},
		{"sequence break backend_timeout", "/sequence-break-backend_timeout", "backend_timeout", []string{"resolve"}},
		{"break only one sequence", "/break-only-one-sequence", "unexpected_status", []string{"resolve2", "resolve1", "refl"}},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()
			h := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://domain.local:8080"+tc.path, nil)
			h.Must(err)

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusBadGateway {
				st.Fatalf("want: %d, got: %d", http.StatusBadGateway, res.StatusCode)
			}

			time.Sleep(time.Millisecond * 200)

			berNames := make(map[string]struct{})
			for _, e := range hook.AllEntries() {
				if e.Data["type"] == "couper_backend" {
					request := e.Data["request"].(logging.Fields)
					berNames[fmt.Sprintf("%s", request["name"])] = struct{}{}
				} else if e.Data["type"] == "couper_access" {
					if e.Data["error_type"] != tc.expectedErrorType {
						st.Errorf("want: %q, got: %q", tc.expectedErrorType, e.Data["error_type"])
					}
				}
			}
			if len(berNames) != len(tc.expBERNames) {
				st.Errorf("number of BE request names want: %d, got: %d", len(tc.expBERNames), len(berNames))
			} else {
				for _, n := range tc.expBERNames {
					if _, ok := berNames[n]; !ok {
						st.Errorf("missing BE request %q", n)
					}
				}
			}
		})
	}
}

func TestEndpointACBufferOptions(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "17_couper.hcl"), helper)
	defer shutdown()

	invalidToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.p_L2kBaXuGvD2AhW5WEheAKLErYXPDR-LKj_dZ5G_XI"
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.6M2CwQMZ-PkeSyREi5scviq0EilhUUSgax6W9TmxmS8"

	urlencoded := func(token string) string {
		return url.Values{"token": []string{token}}.Encode()
	}
	jsonFn := func(token string) string {
		return fmt.Sprintf("{%q: %q}", "token", token)
	}
	plain := func(token string) string {
		return token
	}

	type testcase struct {
		name              string
		path              string
		token             string
		bodyFunc          func(string) string
		contentType       string
		expectedStatus    int
		expectedErrorType string
	}

	for _, tc := range []testcase{
		{"with ac (token in-form_body) and wrong token", "/in-form_body", invalidToken, urlencoded, "application/x-www-form-urlencoded", http.StatusForbidden, "jwt_token_invalid"},
		{"with ac (token in-form_body) and without token", "/in-form_body", "", urlencoded, "application/x-www-form-urlencoded", http.StatusUnauthorized, "jwt_token_missing"},
		{"with ac (token in-form_body) and valid token", "/in-form_body", validToken, urlencoded, "application/x-www-form-urlencoded", http.StatusOK, ""},
		{"with ac (token in-json_body) and wrong token", "/in-json_body", invalidToken, jsonFn, "application/json", http.StatusForbidden, "jwt_token_invalid"},
		{"with ac (token in-json_body) and without token", "/in-json_body", "", jsonFn, "application/json", http.StatusUnauthorized, "jwt_token_missing"},
		{"with ac (token in-json_body) and valid token", "/in-json_body", validToken, jsonFn, "application/json", http.StatusOK, ""},
		{"with ac (token in-body) and wrong token", "/in-body", invalidToken, plain, "text/plain", http.StatusForbidden, "jwt_token_invalid"},
		{"with ac (token in-body) and without token", "/in-body", "", plain, "text/plain", http.StatusUnauthorized, "jwt_token_missing"},
		{"with ac (token in-body) and valid token", "/in-body", validToken, plain, "text/plain", http.StatusOK, ""},
		{"without ac", "/without-ac", "", nil, "text/plain", http.StatusOK, ""},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()
			h := test.New(st)

			body := ""
			if tc.bodyFunc != nil {
				body = tc.bodyFunc(tc.token)
			}
			req, err := http.NewRequest(http.MethodPost, "http://domain.local:8080"+tc.path, strings.NewReader(body))
			h.Must(err)

			req.Header.Set("Content-Type", tc.contentType)
			res, err := client.Do(req)
			h.Must(err)

			_, _ = io.Copy(io.Discard, res.Body)
			h.Must(res.Body.Close())

			if res.StatusCode != tc.expectedStatus {
				st.Errorf("want: %d, got: %d", tc.expectedStatus, res.StatusCode)
			}

			if tc.expectedErrorType != "" {
				for _, e := range hook.AllEntries() {
					if e.Data["type"] != "couper_access" {
						continue
					}
					if e.Data["error_type"] != tc.expectedErrorType {
						st.Errorf("want: %q, got: %v", tc.expectedErrorType, e.Data["error_type"])
					}
				}
			}
		})
	}
}

func TestEndpoint_ReusableProxies(t *testing.T) {
	client := test.NewHTTPClient()
	helper := test.New(t)

	shutdown, hook := newCouper(filepath.Join(testdataPath, "18_couper.hcl"), helper)
	defer shutdown()

	type testCase struct {
		path      string
		name      string
		expStatus int
	}

	for _, tc := range []testCase{
		{"/abcdef", "abcdef", 204},
		{"/reuse", "abcdef", 204},
		{"/default", "default", 200},
		{"/api-abcdef", "abcdef", 204},
		{"/api-reuse", "abcdef", 204},
		{"/api-default", "default", 200},
	} {
		t.Run(tc.path, func(st *testing.T) {
			h := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			h.Must(err)

			hook.Reset()

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("want: %d, got: %d", tc.expStatus, res.StatusCode)
			}

			for _, e := range hook.AllEntries() {
				if e.Data["type"] != "couper_backend" {
					continue
				}

				if name := e.Data["request"].(logging.Fields)["name"]; name != tc.name {
					st.Errorf("want: %s, got: %s", tc.name, name)
				}
			}
		})
	}
}

func TestEndpointWildcardProxyPathWildcard(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/endpoints/21_couper.hcl", test.New(t))
	defer shutdown()

	for _, testcase := range []struct {
		path, expectedPath string
		statusCode         int
	}{
		{"/", "/", http.StatusNotFound},
		{"/anything", "/anything", http.StatusOK},
		{"/a/b", "/a/b", http.StatusNotFound},
		{"/p/", "/pb/", http.StatusNotFound},
		{"/p/a/c", "/pb/a/c", http.StatusNotFound},
	} {
		t.Run(testcase.path[1:], func(st *testing.T) {
			helper := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+testcase.path, nil)
			helper.Must(err)

			hook.Reset()

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != testcase.statusCode {
				st.Errorf("expected status %d, got %d", testcase.statusCode, res.StatusCode)
			}

			b, err := io.ReadAll(res.Body)
			helper.Must(res.Body.Close())
			helper.Must(err)

			type result struct {
				Path string
			}
			r := result{}
			helper.Must(json.Unmarshal(b, &r))

			if testcase.expectedPath != r.Path {
				st.Errorf("Expected path: %q, got: %q", testcase.expectedPath, r.Path)
			}
		})
	}
}
