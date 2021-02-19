package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSOptions_NeedsVary(t *testing.T) {
	tests := []struct {
		name        string
		corsOptions *CORSOptions
		exp         bool
	}{
		{
			"any origin",
			&CORSOptions{AllowedOrigins: []string{"*"}},
			false,
		},
		{
			"one specific origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			true,
		},
		{
			"several specific origins",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com", "http://www.another.host.com"}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			needed := tt.corsOptions.NeedsVary()
			if needed != tt.exp {
				subT.Errorf("Expected %t, got: %t", tt.exp, needed)
			}
		})
	}
}

func TestCORSOptions_AllowsOrigin(t *testing.T) {
	tests := []struct {
		name        string
		corsOptions *CORSOptions
		origin      string
		exp         bool
	}{
		{
			"any origin allowed, specific origin",
			&CORSOptions{AllowedOrigins: []string{"*"}},
			"https://www.example.com",
			true,
		},
		{
			"any origin allowed, *",
			&CORSOptions{AllowedOrigins: []string{"*"}},
			"*",
			true,
		},
		{
			"one specific origin allowed, specific allowed origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			"https://www.example.com",
			true,
		},
		{
			"one specific origin allowed, specific disallowed origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			"http://www.another.host.com",
			false,
		},
		{
			"one specific origin allowed, *",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			"*",
			false,
		},
		{
			"several specific origins allowed, specific origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com", "http://www.another.host.com"}},
			"https://www.example.com",
			true,
		},
		{
			"several specific origins allowed, specific disallowed origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com", "http://www.another.host.com"}},
			"https://www.disallowed.host.org",
			false,
		},
		{
			"several specific origins allowed, *",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com", "http://www.another.host.com"}},
			"*",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			allowed := tt.corsOptions.AllowsOrigin(tt.origin)
			if allowed != tt.exp {
				subT.Errorf("Expected %t, got: %t", tt.exp, allowed)
			}
		})
	}
}

func TestCORSOptions_isCorsRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestHeaders map[string]string
		exp            bool
	}{
		{
			"without Origin",
			map[string]string{},
			false,
		},
		{
			"with Origin",
			map[string]string{"Origin": "https://www.example.com"},
			true,
		},
	}

	cors := &CORS{}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			corsRequest := cors.isCorsRequest(req)
			if corsRequest != tt.exp {
				subT.Errorf("Expected %t, got: %t", tt.exp, corsRequest)
			}
		})
	}
}

func TestCORSOptions_isCorsPreflightRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestHeaders map[string]string
		exp            bool
	}{
		{
			"OPTIONS, without Origin",
			http.MethodOptions,
			map[string]string{},
			false,
		},
		{
			"OPTIONS, with Origin",
			http.MethodOptions,
			map[string]string{"Origin": "https://www.example.com"},
			false,
		},
		{
			"POST, with Origin, with ACRM",
			http.MethodPost,
			map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST"},
			false,
		},
		{
			"POST, with Origin, with ACRH",
			http.MethodPost,
			map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Headers": "Content-Type"},
			false,
		},
		{
			"OPTIONS, with Origin, with ACRM",
			http.MethodOptions,
			map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST"},
			true,
		},
		{
			"OPTIONS, with Origin, with ACRH",
			http.MethodOptions,
			map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Headers": "Content-Type"},
			true,
		},
	}

	cors := &CORS{}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			req := httptest.NewRequest(tt.method, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			corsPfRequest := cors.isCorsPreflightRequest(req)
			if corsPfRequest != tt.exp {
				subT.Errorf("Expected %t, got: %t", tt.exp, corsPfRequest)
			}
		})
	}
}
