package handler

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHealth_Match(t *testing.T) {
	tests := []struct {
		name string
		path string
		req  *http.Request
		want bool
	}{
		{"request w/o health url", healthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/features", nil), false},
		{"request w health url", healthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/healthz", nil), true},
		{"request w health url & query", healthPath, httptest.NewRequest(http.MethodGet, "https://couper.io/healthz?ingress=nginx", nil), true},
		{"request w reconfigured non health url", healthPath + "zz", httptest.NewRequest(http.MethodGet, "https://couper.io/healthz", nil), false},
		{"request w reconfigured health url", healthPath + "zz", httptest.NewRequest(http.MethodGet, "https://couper.io/healthzzz", nil), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Health{
				path: tt.path,
			}
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
	}

	tests := []struct {
		name       string
		fields     fields
		req        *http.Request
		wantStatus int
	}{
		{"healthy check", fields{shutdownCh: make(chan struct{})}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"healthy check /w nil chan", fields{}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"unhealthy check", fields{shutdownCh: make(chan struct{})}, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthCheck(tt.fields.path, tt.fields.shutdownCh)
			rec := httptest.NewRecorder()
			if tt.wantStatus >= 500 {
				close(tt.fields.shutdownCh)
			}
			h.ServeHTTP(rec, tt.req)

			if !rec.Flushed {
				rec.Flush()
			}

			res := rec.Result()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("Expected statusCode: %d, got: %d", tt.wantStatus, res.StatusCode)
			}

			if res.Header.Get("Cache-Control") != "no-store" {
				t.Error("Expected Cache-Control header with 'no-store' value")
			}

			if res.Header.Get("Content-Type") != "text/plain" {
				t.Error("Expected Content-Type header with 'text/plain' value")
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}
			err = res.Body.Close()
			if err != nil {
				t.Error(err)
			}

			if tt.wantStatus == http.StatusOK {
				if string(body) != "healthy" {
					t.Errorf("Expected 'healthy' body content, got %q", string(body))
				}
			} else if string(body) != "server shutting down" {
				t.Errorf("Expected 'server shutting down' body content, got %q", string(body))
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
		want *Health
	}{
		{"/w given path", args{"/myhealth", shutdownChan}, &Health{"/myhealth", shutdownChan}},
		{"/w given path w/o leading slash", args{"myhealth", shutdownChan}, &Health{"/myhealth", shutdownChan}},
		{"w/o given path", args{"", shutdownChan}, &Health{healthPath, shutdownChan}},
		{"w/o given path & chan", args{"", nil}, &Health{healthPath, nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHealthCheck(tt.args.path, tt.args.shutdownCh); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealthCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
