package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
)

func TestProxy_ServeHTTP_Timings(t *testing.T) {
	t.Skip("todo: fixme")
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb proxy settings
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name           string
		options        *ProxyOptions
		req            *http.Request
		expectedStatus int
	}{
		{"with zero timings", &ProxyOptions{Origin: origin.URL}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusNoContent},
		{"with overall timeout", &ProxyOptions{Origin: "http://1.2.3.4/", Timeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusBadGateway},
		{"with connect timeout", &ProxyOptions{Origin: "http://blackhole.webpagetest.org/", ConnectTimeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusBadGateway},
		{"with ttfb timeout", &ProxyOptions{Origin: origin.URL, TTFBTimeout: time.Second}, httptest.NewRequest(http.MethodHead, "http://1.2.3.4/", nil), http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(tt.options, logger.WithContext(nil), eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectedStatus, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					t.Error(log.Message)
				}
			}
		})
	}
}

func TestProxy_ServeHTTP_CORS_PFC(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("from upstream"))
	}))
	defer origin.Close()

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
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(&ProxyOptions{Origin: origin.URL, CORS: tt.corsOptions}, logger.WithContext(nil), eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)

			tt.expectedResponseHeaders["Vary"] = ""
			tt.expectedResponseHeaders["Content-Type"] = ""

			for name, expValue := range tt.expectedResponseHeaders {
				value := rec.HeaderMap.Get(name)
				if value != expValue {
					t.Errorf("Expected %s %s, got: %s", name, expValue, value)
				}
			}

			if rec.Code != http.StatusNoContent {
				t.Errorf("Expected status %d, got: %d", http.StatusNoContent, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					t.Error(log.Message)
				}
			}
		})
	}
}

func TestProxy_ServeHTTP_CORS(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("from upstream"))
	}))
	defer origin.Close()

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
				"Origin":        "https://www.example.com",
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
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(&ProxyOptions{Origin: origin.URL, CORS: tt.corsOptions}, logger.WithContext(context.Background()), eval.NewENVContext(nil))
			if err != nil {
				subT.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)

			for name, expValue := range tt.expectedResponseHeaders {
				value := rec.HeaderMap.Get(name)
				if value != expValue {
					subT.Errorf("%s:\n\tExpected: %s %q, got: %s", tt.name, name, expValue, value)
				}
			}

			if rec.Code != http.StatusOK {
				subT.Errorf("Expected status %d, got: %d", http.StatusOK, rec.Code)
			} else {
				return // no error log for expected codes
			}

			for _, log := range hook.AllEntries() {
				if log.Level >= logrus.ErrorLevel {
					subT.Error(log.Message)
				}
			}
		})
	}
}

func TestProxy_director(t *testing.T) {
	defaultReq := httptest.NewRequest("GET", "http://example.com", nil)

	log, _ := logrustest.NewNullLogger()

	type fields struct {
		evalContext *hcl.EvalContext
		log         *logrus.Entry
		options     *ProxyOptions
	}

	emptyOptions := []hcl.Body{hcl.EmptyBody()}

	tests := []struct {
		name   string
		fields fields
		req    *http.Request
		expReq *http.Request
	}{
		{"proxy url settings", fields{eval.NewENVContext(nil), log.WithContext(nil), &ProxyOptions{Origin: "http://1.2.3.4", Context: emptyOptions}}, defaultReq, httptest.NewRequest("GET", "http://1.2.3.4", nil)},
		{"proxy url settings w/hostname", fields{eval.NewENVContext(nil), log.WithContext(nil), &ProxyOptions{Origin: "http://1.2.3.4", Hostname: "couper.io", Context: emptyOptions}}, defaultReq, httptest.NewRequest("GET", "http://couper.io", nil)},
		{"proxy url settings w/wildcard ctx", fields{eval.NewENVContext(nil), log.WithContext(nil), &ProxyOptions{Origin: "http://1.2.3.4", Hostname: "couper.io", Path: "/**", Context: emptyOptions}}, defaultReq.WithContext(context.WithValue(defaultReq.Context(), request.Wildcard, "/hans")), httptest.NewRequest("GET", "http://couper.io/hans", nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProxy(tt.fields.options, tt.fields.log, tt.fields.evalContext)
			if err != nil {
				t.Fatal(err)
			}
			proxy := p.(*Proxy)
			proxy.director(tt.req)

			if tt.fields.options.Hostname != "" && tt.fields.options.Hostname != tt.expReq.Host {
				t.Errorf("expected same host value, want: %q, got: %q", tt.fields.options.Hostname, tt.expReq.Host)
			} else if tt.fields.options.Hostname == "" && tt.req.Host != tt.expReq.Host {
				t.Error("expected a configured request host")
			}

			if tt.req.URL.Path != tt.expReq.URL.Path {
				t.Errorf("expected path: %q, got: %q", tt.expReq.URL.Path, tt.req.URL.Path)
			}
		})
	}
}

func TestProxy_modifyResponse(t *testing.T) {
	type fields struct {
		evalContext *hcl.EvalContext
		log         *logrus.Entry
		options     *ProxyOptions
	}

	type args struct {
		res *http.Response
	}

	type testCase struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}

	tests := []testCase{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProxy(tt.fields.options, tt.fields.log, tt.fields.evalContext)
			if err != nil {
				t.Fatal(err)
			}
			// TODO: test me
		})
	}
}

func TestProxy_isCredentialed(t *testing.T) {
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

			p := &Proxy{}
			credentialed := p.isCredentialed(req.Header)
			if (credentialed != tt.exp) {
				t.Errorf("expected: %t, got: %t", tt.exp, credentialed)
			}
		})
	}
}

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
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			corsRequest := isCorsRequest(req)
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
			"OPTIONS, without Origin, with ACRM",
			http.MethodOptions,
			map[string]string{"Access-Control-Request-Method": "POST"},
			false,
		},
		{
			"OPTIONS, without Origin, with ACRH",
			http.MethodOptions,
			map[string]string{"Access-Control-Request-Headers": "Content-Type"},
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
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			req := httptest.NewRequest(tt.method, "http://1.2.3.4/", nil)
			for name, value := range tt.requestHeaders {
				req.Header.Set(name, value)
			}

			corsPfRequest := isCorsPreflightRequest(req)
			if corsPfRequest != tt.exp {
				subT.Errorf("Expected %t, got: %t", tt.exp, corsPfRequest)
			}
		})
	}
}
