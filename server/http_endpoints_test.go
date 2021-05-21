package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
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

			resBytes, err := ioutil.ReadAll(res.Body)
			helper.Must(err)

			_ = res.Body.Close()

			var jsonResult expectation
			err = json.Unmarshal(resBytes, &jsonResult)
			if err != nil {
				t.Errorf("unmarshal json: %v: got:\n%s", err, string(resBytes))
			}

			if !reflect.DeepEqual(jsonResult, tc.exp) {
				t.Errorf("\nwant: \n%#v\ngot: \n%#v\npayload:\n%s", tc.exp, jsonResult, string(resBytes))
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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

	resBytes, err := ioutil.ReadAll(res.Body)
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
