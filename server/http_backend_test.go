package server_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/logging"
)

func TestBackend_MaxConnections(t *testing.T) {
	helper := test.New(t)

	const reqCount = 3
	lastSeen := map[string]string{}
	lastSeenMu := sync.Mutex{}
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		lastSeenMu.Lock()
		defer lastSeenMu.Unlock()

		if lastSeen[r.URL.Path] != "" && lastSeen[r.URL.Path] != r.RemoteAddr {
			t.Errorf("expected same remote addr for path: %q", r.URL.Path)
			rw.WriteHeader(http.StatusInternalServerError)
		} else {
			rw.WriteHeader(http.StatusNoContent)
		}
		lastSeen[r.URL.Path] = r.RemoteAddr
	}))

	defer origin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/03_couper.hcl", helper, map[string]interface{}{
		"origin": origin.URL,
	})
	defer shutdown()

	paths := []string{
		"/",
		"/be",
		"/fake-sequence",
	}

	originWait := sync.WaitGroup{}
	originWait.Add(len(paths) * reqCount)
	waitForCh := make(chan struct{})

	client := test.NewHTTPClient()

	for _, clientPath := range paths {
		for i := 0; i < reqCount; i++ {
			go func(path string) {
				req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+path, nil)
				<-waitForCh
				res, err := client.Do(req)
				helper.Must(err)

				if res.StatusCode != http.StatusNoContent {
					t.Errorf("want: 204, got %d", res.StatusCode)
				}

				originWait.Done()
			}(clientPath)
		}
	}

	close(waitForCh)
	originWait.Wait()
}

func TestBackend_MaxConnections_BodyClose(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2) // always delay, ensures every req hit runs into max_conns issue

		rw.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(r.URL)
		helper.Must(err)
		_, err = rw.Write(b)
		helper.Must(err)
	}))

	defer origin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/04_couper.hcl", helper,
		map[string]interface{}{
			"origin": origin.URL,
		})
	defer shutdown()

	client := test.NewHTTPClient()

	paths := []string{
		"/",
		"/named",
		"/default",
		"/default2",
		"/ws",
	}

	for _, p := range paths {
		deadline, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+p, nil)
		res, err := client.Do(req.WithContext(deadline))
		cancel()
		helper.Must(err)

		if res.StatusCode != http.StatusOK {
			t.Errorf("want: 200, got %d", res.StatusCode)
		}

		_, err = io.Copy(io.Discard, res.Body)
		helper.Must(err)

		helper.Must(res.Body.Close())
	}
}

// TestBackend_WithoutOrigin expects the listed errors to ensure no host from the client-request
// leaks into the backend structure for connecting to the origin.
func TestBackend_WithoutOrigin(t *testing.T) {
	helper := test.New(t)
	shutdown, hook := newCouper("testdata/integration/backends/01_couper.hcl", helper)
	defer shutdown()

	client := test.NewHTTPClient()

	for _, tc := range []struct {
		path    string
		message string
	}{
		{"/proxy/backend-path", `configuration error: anonymous_6_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/proxy/url", `configuration error: anonymous_15_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/request/backend-path", `configuration error: anonymous_28_15: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/request/url", `configuration error: anonymous_37_15: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
	} {
		t.Run(tc.path, func(st *testing.T) {
			hook.Reset()

			h := test.New(st)
			req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+tc.path, nil)
			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				st.Errorf("want: 500, got %d", res.StatusCode)
			}

			for _, e := range hook.AllEntries() {
				if e.Level != logrus.ErrorLevel {
					continue
				}

				if e.Message != tc.message {
					st.Errorf("\nwant: %q\ngot:  %q\n", tc.message, e.Message)
				}
			}

		})

	}
}

func TestBackend_LogResponseBytes(t *testing.T) {
	helper := test.New(t)

	var writtenBytes int64
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(r.URL)
		helper.Must(err)

		if r.Header.Get("Accept-Encoding") == "gzip" {
			buf := &bytes.Buffer{}
			gw := gzip.NewWriter(buf)
			_, zerr := gw.Write(b)
			helper.Must(zerr)
			helper.Must(gw.Close())

			atomic.StoreInt64(&writtenBytes, int64(buf.Len()))

			_, err = io.Copy(rw, buf)
			helper.Must(err)
		} else {
			n, werr := rw.Write(b)
			helper.Must(werr)
			atomic.StoreInt64(&writtenBytes, int64(n))
		}
	}))

	defer origin.Close()

	shutdown, hook := newCouperWithTemplate("testdata/integration/backends/05_couper.hcl", helper,
		map[string]interface{}{
			"origin": origin.URL,
		})
	defer shutdown()

	client := test.NewHTTPClient()

	cases := []struct {
		accept string
		path   string
	}{
		{path: "/"},
		{accept: "gzip", path: "/zipped"},
	}

	for _, tc := range cases {
		hook.Reset()

		deadline, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+tc.path, nil)

		if tc.accept != "" {
			req.Header.Set("Accept-Encoding", tc.accept)
		}

		res, err := client.Do(req.WithContext(deadline))
		cancel()
		helper.Must(err)

		if res.StatusCode != http.StatusOK {
			t.Errorf("want: 200, got %d", res.StatusCode)
		}

		_, err = io.Copy(io.Discard, res.Body)
		helper.Must(err)

		helper.Must(res.Body.Close())

		var seen bool
		for _, e := range hook.AllEntries() {
			if e.Data["type"] != "couper_backend" {
				continue
			}

			seen = true

			response, ok := e.Data["response"]
			if !ok {
				t.Error("expected response log field")
			}

			bytesValue, bok := response.(logging.Fields)["bytes"]
			if !bok {
				t.Error("expected response.bytes log field")
			}

			expectedBytes := atomic.LoadInt64(&writtenBytes)
			if bytesValue.(int64) != expectedBytes {
				t.Errorf("bytes differs: want: %d, got: %d", expectedBytes, bytesValue)
			}
		}

		if !seen {
			t.Error("expected upstream log")
		}
	}
}

func TestBackend_Unhealthy(t *testing.T) {
	helper := test.New(t)

	var unhealthy int64
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if counter := r.Header.Get("Counter"); counter != "" {
			c, _ := strconv.Atoi(counter)
			if c > 2 {
				atomic.StoreInt64(&unhealthy, 1)
			}
		}
		if atomic.LoadInt64(&unhealthy) == 1 {
			rw.WriteHeader(http.StatusConflict)
		} else {
			rw.WriteHeader(http.StatusNoContent)
			time.Sleep(time.Second / 3)
		}
	}))

	defer origin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/06_couper.hcl", helper,
		map[string]interface{}{
			"origin": origin.URL,
		})
	defer shutdown()

	client := test.NewHTTPClient()

	type testcase struct {
		path      string
		expStatus int
	}

	for i, tc := range []testcase{
		{"/anon", http.StatusNoContent},
		{"/ref", http.StatusNoContent},
		{"/catch", http.StatusNoContent},
		// server switched resp status-code -> unhealthy
		{"/anon", http.StatusConflict}, // always healthy
		{"/ref", http.StatusBadGateway},
		{"/catch", http.StatusTeapot},
	} {
		t.Run(tc.path, func(st *testing.T) {
			h := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+tc.path, nil)
			h.Must(err)
			req.Header.Set("Counter", strconv.Itoa(i))
			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("want status %d, got: %d", tc.expStatus, res.StatusCode)
			}
		})
	}
}

func TestBackend_Oauth2_TokenEndpoint(t *testing.T) {
	helper := test.New(t)

	requestCount := 0
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusUnauthorized)
		_, werr := rw.Write([]byte(`{"path": "` + r.URL.Path + `"}`))
		requestCount++
		helper.Must(werr)
	}))
	defer origin.Close()

	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, werr := rw.Write([]byte(`{
          	"access_token": "my-token",
          	"expires_in": 120
		}`))
		helper.Must(werr)
	}))
	defer tokenEndpoint.Close()

	retries := 3
	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/07_couper.hcl", helper,
		map[string]interface{}{
			"origin":         origin.URL,
			"token_endpoint": tokenEndpoint.URL,
			"retries":        retries,
		})
	defer shutdown()

	client := test.NewHTTPClient()

	req, err := http.NewRequest(http.MethodGet, "http://couper.dev:8080/test-path", nil)
	helper.Must(err)
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("want status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("want json content-type")
		return
	}

	type result struct {
		Path string
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)
	helper.Must(res.Body.Close())

	r := &result{}
	helper.Must(json.Unmarshal(b, r))

	if r.Path != "/test-path" {
		t.Errorf("path property want: %q, got: %q", "/test-path", r.Path)
	}

	if requestCount != retries+1 {
		t.Errorf("unexpected number of requests, want: %d, got: %d", retries+1, requestCount)
	}
}
