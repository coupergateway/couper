package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/coupergateway/couper/internal/test"
)

// TestHTTPServer_BackendResponseTrailers ensures HTTP/2 backend response trailers
// (e.g. the gRPC grpc-status trailer) are forwarded to the client instead of being
// dropped. See issue #968.
func TestHTTPServer_BackendResponseTrailers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		expect  map[string]string
	}{
		{
			// Mirrors the reproducer in issue #968: an announced trailer set
			// after the body plus an unannounced (gRPC-style) one.
			name: "announced and unannounced",
			handler: func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Set("Content-Type", "application/grpc")
				rw.Header().Set(http.TrailerPrefix+"Grpc-Status", "0") // unannounced
				rw.Header().Set("Trailer", "X-Announced")              // announced
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write([]byte("body\n"))
				rw.Header().Set("X-Announced", "after-body")
			},
			expect: map[string]string{"Grpc-Status": "0", "X-Announced": "after-body"},
		},
		{
			// Real gRPC: the grpc-status trailer is unannounced and only
			// appears after a (here large, streamed) body, so the response
			// header has already been flushed before it is known.
			name: "unannounced only with large body",
			handler: func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Set("Content-Type", "application/grpc")
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(bytes.Repeat([]byte("x"), 4096))
				rw.Header().Set(http.TrailerPrefix+"Grpc-Status", "0")
			},
			expect: map[string]string{"Grpc-Status": "0"},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)
			client := newClient()

			// HTTP/2 (TLS) backend; Couper proxies it to an HTTP/1.1 client.
			h2 := httptest.NewUnstartedServer(tc.handler)
			h2.EnableHTTP2 = true
			h2.StartTLS()
			defer h2.Close()

			shutdown, _, err := newCouperWithTemplate(
				"testdata/integration/trailers/01_couper.hcl",
				helper,
				map[string]interface{}{"origin": h2.URL},
			)
			helper.Must(err)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/x", nil)
			helper.Must(err)
			res, err := client.Do(req)
			helper.Must(err)

			_, _ = io.Copy(io.Discard, res.Body) // drain so trailers populate
			helper.Must(res.Body.Close())

			for k, want := range tc.expect {
				if got := res.Trailer.Get(k); got != want {
					subT.Errorf("%s trailer: want %q, got %q (all: %v)", k, want, got, res.Trailer)
				}
			}
		})
	}
}

// TestHTTPServer_HTTP2BackendWithoutTrailers guards the Content-Length removal that
// trailer forwarding needs: it applies to every HTTP/2 backend response, so ordinary
// proxying, json_body evaluation and empty responses must stay intact. A small
// response keeps a Content-Length because net/http recomputes it after the handler
// wrote the complete body; only a body that outgrows the write buffer switches the
// HTTP/1.1 client to chunked framing.
func TestHTTPServer_HTTP2BackendWithoutTrailers(t *testing.T) {
	const largeBodyLen = 8192

	helper := test.New(t)
	client := newClient()

	jsonBody := func(proto string) []byte {
		return []byte(`{"name":"couper","proto":"` + proto + `"}`)
	}

	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/nobody"):
			rw.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(req.URL.Path, "/large"):
			rw.Header().Set("Content-Type", "application/octet-stream")
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(bytes.Repeat([]byte("x"), largeBodyLen))
		default:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(jsonBody(req.Proto))
		}
	}))
	origin.EnableHTTP2 = true
	origin.StartTLS()
	defer origin.Close()

	shutdown, _, err := newCouperWithTemplate(
		"testdata/integration/trailers/02_couper.hcl",
		helper,
		map[string]interface{}{"origin": origin.URL},
	)
	helper.Must(err)
	defer shutdown()

	get := func(subT *testing.T, path string) (*http.Response, []byte) {
		subT.Helper()
		h := test.New(subT)

		req, rErr := http.NewRequest(http.MethodGet, "http://example.com:8080"+path, nil)
		h.Must(rErr)
		res, rErr := client.Do(req)
		h.Must(rErr)

		body, rErr := io.ReadAll(res.Body)
		h.Must(rErr)
		h.Must(res.Body.Close())

		if len(res.Trailer) != 0 {
			subT.Errorf("trailers: want none, got %v", res.Trailer)
		}
		if tr := res.Header.Get("Trailer"); tr != "" {
			subT.Errorf("trailer header: want none, got %q", tr)
		}

		return res, body
	}

	t.Run("json body forwarded", func(subT *testing.T) {
		res, body := get(subT, "/h2/json")

		if res.StatusCode != http.StatusOK {
			subT.Errorf("status: want %d, got %d", http.StatusOK, res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); ct != "application/json" {
			subT.Errorf("content-type: want application/json, got %q", ct)
		}

		var parsed map[string]string
		if jErr := json.Unmarshal(body, &parsed); jErr != nil {
			subT.Fatalf("client cannot parse the forwarded body %q: %v", body, jErr)
		}
		if parsed["proto"] != "HTTP/2.0" {
			subT.Errorf("origin protocol: want HTTP/2.0, got %q", parsed["proto"])
		}
		if !bytes.Equal(body, jsonBody("HTTP/2.0")) {
			subT.Errorf("body: want %q, got %q", jsonBody("HTTP/2.0"), body)
		}
		if cl := res.Header.Get("Content-Length"); cl != strconv.Itoa(len(body)) {
			subT.Errorf("content-length: want %d, got %q", len(body), cl)
		}
		if len(res.TransferEncoding) != 0 {
			subT.Errorf("transfer-encoding: want none, got %v", res.TransferEncoding)
		}
	})

	t.Run("large body switches to chunked", func(subT *testing.T) {
		res, body := get(subT, "/h2/large")

		if len(body) != largeBodyLen {
			subT.Errorf("body length: want %d, got %d", largeBodyLen, len(body))
		}
		if cl := res.Header.Get("Content-Length"); cl != "" {
			subT.Errorf("content-length: want it removed, got %q", cl)
		}
		if len(res.TransferEncoding) != 1 || res.TransferEncoding[0] != "chunked" {
			subT.Errorf("transfer-encoding: want [chunked], got %v", res.TransferEncoding)
		}
	})

	t.Run("json_body evaluated and body still forwarded", func(subT *testing.T) {
		res, body := get(subT, "/h2eval/json")

		if got := res.Header.Get("X-From-Json-Body"); got != "couper" {
			subT.Errorf("x-from-json-body: want %q, got %q", "couper", got)
		}
		if !bytes.Equal(body, jsonBody("HTTP/2.0")) {
			subT.Errorf("body: want %q, got %q", jsonBody("HTTP/2.0"), body)
		}
	})

	t.Run("empty response", func(subT *testing.T) {
		res, body := get(subT, "/h2/nobody")

		if res.StatusCode != http.StatusNoContent {
			subT.Errorf("status: want %d, got %d", http.StatusNoContent, res.StatusCode)
		}
		if len(body) != 0 {
			subT.Errorf("body: want empty, got %q", body)
		}
	})

	t.Run("http1 backend keeps content-length", func(subT *testing.T) {
		res, body := get(subT, "/h1/json")

		if !bytes.Equal(body, jsonBody("HTTP/1.1")) {
			subT.Errorf("body: want %q, got %q", jsonBody("HTTP/1.1"), body)
		}
		if cl := res.Header.Get("Content-Length"); cl != strconv.Itoa(len(body)) {
			subT.Errorf("content-length: want %d, got %q", len(body), cl)
		}
		if len(res.TransferEncoding) != 0 {
			subT.Errorf("transfer-encoding: want none, got %v", res.TransferEncoding)
		}
	})
}
