package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

const testdataPath = "testdata/endpoints"

func TestEndpoints_ProxyReqRes(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "01_couper.hcl"), helper)
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

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, given %d", res.StatusCode)
	}

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "1616" {
		t.Errorf("Expected body 1616, given %s", resBytes)
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

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)
	var seenCh, tokenSeenCh chan struct{}

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth2" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)
			close(tokenSeenCh)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/resource" {
			if req.Header.Get("Authorization") == "Bearer abcdef0123456789" {
				rw.WriteHeader(http.StatusNoContent)
				close(seenCh)
				return
			}
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	defer ResourceOrigin.Close()

	confPath := "testdata/endpoints/04_couper.hcl"
	shutdown, hook := newCouper(confPath, test.New(t))
	defer func() {
		if t.Failed() {
			for _, e := range hook.Entries {
				println(e.String())
			}
		}
		shutdown()
	}()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	req.Header.Set("X-Token-Endpoint", oauthOrigin.URL)
	req.Header.Set("X-Origin", ResourceOrigin.URL)

	for _, p := range []string{"/", "/2nd"} {
		hook.Reset()

		seenCh = make(chan struct{})
		tokenSeenCh = make(chan struct{})

		req.URL.Path = p
		res, err := newClient().Do(req)
		helper.Must(err)

		if res.StatusCode != http.StatusNoContent {
			t.Errorf("expected status NoContent, got: %d", res.StatusCode)
			return
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-timer.C:
			t.Error("OAuth2 request failed")
		case <-tokenSeenCh:
			<-seenCh
		}
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
