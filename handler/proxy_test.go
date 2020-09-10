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
		req                     *http.Request
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{"with ACRM", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST"},  map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Methods": "POST", "Access-Control-Allow-Headers": "", "Access-Control-Allow-Credentials": "", "Access-Control-Max-Age": ""}},
		{"with ACRH", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Headers": "X-Foo, X-Bar"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Methods": "", "Access-Control-Allow-Headers": "X-Foo, X-Bar", "Access-Control-Allow-Credentials": "", "Access-Control-Max-Age": ""}},
		{"with ACRM, ACRH", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "X-Foo, X-Bar"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Methods": "POST", "Access-Control-Allow-Headers": "X-Foo, X-Bar", "Access-Control-Allow-Credentials": "", "Access-Control-Max-Age": ""}},
		{"with ACRM, credentials", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}, AllowCredentials:true}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Methods": "POST", "Access-Control-Allow-Headers": "", "Access-Control-Allow-Credentials": "true", "Access-Control-Max-Age": ""}},
		{"with ACRM, max-age", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"},MaxAge:"3600"}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Access-Control-Request-Method": "POST"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Methods": "POST", "Access-Control-Allow-Headers": "", "Access-Control-Allow-Credentials": "", "Access-Control-Max-Age": "3600"}},
		{"origin mismatch", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.org", "Access-Control-Request-Method": "POST"}, map[string]string{"Access-Control-Allow-Origin": "", "Access-Control-Allow-Methods": "", "Access-Control-Allow-Headers": "", "Access-Control-Allow-Credentials": "", "Access-Control-Max-Age": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(&ProxyOptions{Origin: origin.URL, CORS: tt.corsOptions}, logger.WithContext(nil), eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			for name, value := range tt.requestHeaders {
				tt.req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.req)

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
		req                     *http.Request
		requestHeaders          map[string]string
		expectedResponseHeaders map[string]string
	}{
		{"specific origin", &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Credentials": "", "Vary": "Origin"}},
		{"any origin", &CORSOptions{AllowedOrigins:[]string{"*"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com"}, map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Credentials": "", "Vary": ""}},
		{"any origin, cookie credentials", &CORSOptions{AllowedOrigins:[]string{"*"}, AllowCredentials:true}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Cookie": "a=b"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Credentials": "true", "Vary": ""}},
		{"any origin, auth credentials", &CORSOptions{AllowedOrigins:[]string{"*"}, AllowCredentials:true}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), map[string]string{"Origin": "https://www.example.com", "Authorization": "Basic oertnbin"}, map[string]string{"Access-Control-Allow-Origin": "https://www.example.com", "Access-Control-Allow-Credentials": "true", "Vary": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(&ProxyOptions{Origin: origin.URL, CORS: tt.corsOptions}, logger.WithContext(nil), eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			for name, value := range tt.requestHeaders {
				tt.req.Header.Set(name, value)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.req)

			tt.expectedResponseHeaders["Access-Control-Allow-Methods"] = ""
			tt.expectedResponseHeaders["Access-Control-Allow-Headers"] = ""
			tt.expectedResponseHeaders["Access-Control-Max-Age"] = ""
			tt.expectedResponseHeaders["Content-Type"] = "text/plain"

			for name, expValue := range tt.expectedResponseHeaders {
				value := rec.HeaderMap.Get(name)
				if value != expValue {
					t.Errorf("Expected %s %s, got: %s", name, expValue, value)
				}
			}

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got: %d", http.StatusOK, rec.Code)
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
