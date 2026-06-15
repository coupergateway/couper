package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
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
