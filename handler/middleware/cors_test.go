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

func TestCORS_ServeHTTP(t *testing.T) {
	upstreamHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte("from upstream"))
		if err != nil {
			t.Error(err)
		}
	})

	tests := []struct {
		name                    string
		corsOptions             *CORSOptions
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{
			"specific origin",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "Origin",
			},
		},
		{
			"specific origins",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com", "https://example.com"}},
			map[string]string{
				"Origin": "https://example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://example.com",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "Origin",
			},
		},
		{
			"any origin",
			&CORSOptions{AllowedOrigins: []string{"*"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "",
			},
		},
		{
			"any and specific origin",
			&CORSOptions{AllowedOrigins: []string{"https://example.com", "https://www.example.com", "*"}},
			map[string]string{
				"Origin": "https://www.example.com",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "",
				"Vary":                             "",
			},
		},
		{
			"specific origin, cookie credentials",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin": "https://www.example.com",
				"Cookie": "a=b",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			"specific origin, auth credentials",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin":        "https://www.example.com",
				"Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			"any origin, cookie credentials",
			&CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin": "https://www.example.com",
				"Cookie": "a=b",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
		{
			"any origin, auth credentials",
			&CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin":        "https://www.example.com",
				"Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
		{
			"any origin, proxy auth credentials",
			&CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true},
			map[string]string{
				"Origin":              "https://www.example.com",
				"Proxy-Authorization": "Basic oertnbin",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			corsHandler := NewCORSHandler(tt.corsOptions)

			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			rec := httptest.NewRecorder()
			corsHandler.ServeNextHTTP(rec, upstreamHandler, req)

			if !rec.Flushed {
				rec.Flush()
			}

			res := rec.Result()

			for name, expValue := range tt.expectedResponseHeaders {
				value := res.Header.Get(name)
				if value != expValue {
					subT.Errorf("%s:\n\tExpected: %s %q, got: %s", tt.name, name, expValue, value)
				}
			}

			if rec.Code != http.StatusOK {
				subT.Errorf("Expected status %d, got: %d", http.StatusOK, rec.Code)
			} else {
				return // no error log for expected codes
			}
		})
	}
}

func TestProxy_ServeHTTP_CORS_PFC(t *testing.T) {
	upstreamHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte("from upstream"))
		if err != nil {
			t.Error(err)
		}
	})

	tests := []struct {
		name                    string
		corsOptions             *CORSOptions
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{
			"with ACRM",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRH",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                         "https://www.example.com",
				"Access-Control-Request-Headers": "X-Foo, X-Bar",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "X-Foo, X-Bar",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, ACRH",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                         "https://www.example.com",
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "X-Foo, X-Bar",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "X-Foo, X-Bar",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, credentials",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, AllowCredentials: true},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			"with ACRM, max-age",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}, MaxAge: "3600"},
			map[string]string{
				"Origin":                        "https://www.example.com",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "https://www.example.com",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "3600",
			},
		},
		{
			"origin mismatch",
			&CORSOptions{AllowedOrigins: []string{"https://www.example.com"}},
			map[string]string{
				"Origin":                        "https://www.example.org",
				"Access-Control-Request-Method": "POST",
			},
			map[string]string{
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Max-Age":           "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corsHandler := NewCORSHandler(tt.corsOptions)

			req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			rec := httptest.NewRecorder()
			corsHandler.ServeNextHTTP(rec, upstreamHandler, req)

			if !rec.Flushed {
				rec.Flush()
			}

			res := rec.Result()

			tt.expectedResponseHeaders["Vary"] = ""
			tt.expectedResponseHeaders["Content-Type"] = ""

			for name, expValue := range tt.expectedResponseHeaders {
				value := res.Header.Get(name)
				if value != expValue {
					t.Errorf("Expected %s %s, got: %s", name, expValue, value)
				}
			}

			if rec.Code != http.StatusNoContent {
				t.Errorf("Expected status %d, got: %d", http.StatusNoContent, rec.Code)
			} else {
				return // no error log for expected codes
			}
		})
	}
}
