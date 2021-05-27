package handler_test

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/avenga/couper/server/writer"

	"github.com/avenga/couper/handler"
)

func TestHealth_Match(t *testing.T) {
	tests := []struct {
		name string
		path string
		req  *http.Request
		want bool
	}{
		{"request w/o health url", handler.DefaultHealthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/features", nil), false},
		{"request w health url", handler.DefaultHealthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/healthz", nil), true},
		{"request w health url & query", handler.DefaultHealthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/healthz?ingress=nginx", nil), true},
		{"request w reconfigured non health url", handler.DefaultHealthPath + "zz", httptest.NewRequest(http.MethodGet, "https://couper.io/healthz", nil), false},
		{"request w reconfigured health url", handler.DefaultHealthPath + "zz", httptest.NewRequest(http.MethodGet, "https://couper.io/healthzzz", nil), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewHealthCheck(tt.path, nil)
			if got := h.Match(tt.req); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealth_ServeHTTP(t *testing.T) {
	type fields struct {
		path       string
		shutdownCh chan struct{}
		gzip       bool
	}

	tests := []struct {
		name       string
		fields     fields
		req        *http.Request
		wantStatus int
	}{
		{"healthy check", fields{shutdownCh: make(chan struct{})}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"healthy check /w nil chan", fields{}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"healthy check /w gzip", fields{shutdownCh: make(chan struct{}), gzip: true}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"unhealthy check", fields{shutdownCh: make(chan struct{})}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := handler.NewHealthCheck(tt.fields.path, tt.fields.shutdownCh)

			if tt.fields.gzip {
				tt.req.Header.Set("Accept-Encoding", "gzip")
			}

			rec := httptest.NewRecorder()
			gw := writer.NewGzipWriter(rec, tt.req.Header)
			rw := writer.NewResponseWriter(gw, "")
			if tt.wantStatus >= 500 {
				close(tt.fields.shutdownCh)
			}

			h.ServeHTTP(rw, tt.req)
			_ = gw.Close()

			rec.Flush()
			res := rec.Result()

			if rw.StatusCode() != tt.wantStatus {
				subT.Errorf("Expected statusCode: %d, got: %d", tt.wantStatus, rw.StatusCode())
			}

			if rw.Header().Get("Cache-Control") != "no-store" {
				subT.Error("Expected Cache-Control header with 'no-store' value")
			}

			if rw.Header().Get("Content-Type") != "text/plain" {
				subT.Error("Expected Content-Type header with 'text/plain' value")
			}

			if tt.fields.gzip && rw.Header().Get("Content-Encoding") != "gzip" {
				subT.Error("Expected gzip response")
			}
			body := res.Body
			if tt.fields.gzip {
				b, err := gzip.NewReader(body)
				if err != nil {
					subT.Error(err)
					return
				}
				body = b
			}
			bytes, err := ioutil.ReadAll(body)
			if err != nil {
				subT.Error(err)
			}

			if tt.wantStatus == http.StatusOK {
				if string(bytes) != "healthy" {
					subT.Errorf("Expected 'healthy' body content, got %q", string(bytes))
				}
			} else if string(bytes) != "server shutting down" {
				subT.Errorf("Expected 'server shutting down' body content, got %q", string(bytes))
			}
		})
	}
}

func TestNewHealthCheck(t *testing.T) {
	type args struct {
		path       string
		shutdownCh chan struct{}
	}

	shutdownChan := make(chan struct{})

	tests := []struct {
		name string
		args args
		want *handler.Health
	}{
		{"/w given path", args{"/myhealth", shutdownChan}, handler.NewHealthCheck("/myhealth", shutdownChan)},
		{"/w given path w/o leading slash", args{"myhealth", shutdownChan}, handler.NewHealthCheck("/myhealth", shutdownChan)},
		{"w/o given path", args{"", shutdownChan}, handler.NewHealthCheck(handler.DefaultHealthPath, shutdownChan)},
		{"w/o given path & chan", args{"", nil}, handler.NewHealthCheck(handler.DefaultHealthPath, nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.NewHealthCheck(tt.args.path, tt.args.shutdownCh); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealthCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
