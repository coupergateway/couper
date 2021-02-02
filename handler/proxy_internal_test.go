package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_IsCredentialed(t *testing.T) {
	type testCase struct {
		name           string
		requestHeaders map[string]string
		exp            bool
	}

	tests := []testCase{
		{
			"Cookie",
			map[string]string{"Cookie": "a=b"},
			true,
		},
		{
			"Authorization",
			map[string]string{"Authorization": "Basic qeinbqtpoib"},
			true,
		},
		{
			"Proxy-Authorization",
			map[string]string{"Proxy-Authorization": "Basic qeinbqtpoib"},
			true,
		},
		{
			"Not credentialed",
			map[string]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			credentialed := isCredentialed(req.Header)
			if credentialed != tt.exp {
				t.Errorf("expected: %t, got: %t", tt.exp, credentialed)
			}
		})
	}
}
