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

func TestProxy_ServeHTTP_CORS(t *testing.T) {
	// t.Skip("todo: fixme")
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb proxy settings
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name                  string
		options               *ProxyOptions
		req                   *http.Request
		origin                string
		acrm                  string
		acrh                  string
		expectedStatus        int
		expectedContentLength string
		expectedACAO          string
		expectedACAM          string
		expectedACAH          string
		expectedACAC          bool
		expectedACMA          string
		expectedVary          string
	}{
		{"preflight check: ACRM", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.com", "POST", "", http.StatusNoContent, "0", "https://www.example.com", "POST", "", false, "", ""},
		{"preflight check: ACRH", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.com", "", "X-Foo, X-Bar", http.StatusNoContent, "0", "https://www.example.com", "", "X-Foo, X-Bar", false, "", ""},
		{"preflight check: ACRM, ACRH", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.com", "POST", "X-Foo, X-Bar", http.StatusNoContent, "0", "https://www.example.com", "POST", "X-Foo, X-Bar", false, "", ""},
		{"preflight check: ACRM, credentials", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}, AllowCredentials:true}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.com", "POST", "", http.StatusNoContent, "0", "https://www.example.com", "POST", "", true, "", ""},
		{"preflight check: ACRM, max-age", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"},MaxAge:"3600"}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.com", "POST", "", http.StatusNoContent, "0", "https://www.example.com", "POST", "", false, "3600", ""},
		{"preflight check: origin mismatch", &ProxyOptions{Origin: origin.URL, CORS: &CORSOptions{AllowedOrigins:[]string{"https://www.example.com"}}}, httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil), "https://www.example.org", "POST", "", http.StatusNoContent, "", "", "", "", false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			p, err := NewProxy(tt.options, logger.WithContext(nil), eval.NewENVContext(nil))
			if err != nil {
				t.Fatal(err)
			}

			tt.req.Header.Set("Origin", tt.origin)
			if tt.acrm != "" {
				tt.req.Header.Set("Access-Control-Request-Method", tt.acrm)
			}
			if tt.acrh != "" {
				tt.req.Header.Set("Access-Control-Request-Headers", tt.acrh)
			}

			hook.Reset()
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.req)

			acao := rec.HeaderMap.Get("Access-Control-Allow-Origin")
            if acao != tt.expectedACAO {
				t.Errorf("Expected Access-Control-Allow-Origin %s, got: %s", tt.expectedACAO, acao)
			}

			acam := rec.HeaderMap.Get("Access-Control-Allow-Methods")
            if acam != tt.expectedACAM {
				t.Errorf("Expected Access-Control-Allow-Methods %s, got: %s", tt.expectedACAM, acam)
			}

			acah := rec.HeaderMap.Get("Access-Control-Allow-Headers")
            if acah != tt.expectedACAH {
				t.Errorf("Expected Access-Control-Allow-Headers %s, got: %s", tt.expectedACAH, acah)
			}

			acac := rec.HeaderMap.Get("Access-Control-Allow-Credentials")
			if tt.expectedACAC {
				if acac != "true" {
					t.Errorf("Expected Access-Control-Allow-Credentials %s, got: %s", "true", acac)
				}
			} else {
				if acac != "" {
					t.Errorf("Expected Access-Control-Allow-Credentials %s, got: %s", "", acac)
				}
			}

			acma := rec.HeaderMap.Get("Access-Control-Max-Age")
            if acma != tt.expectedACMA {
				t.Errorf("Expected Access-Control-Max-Age %s, got: %s", tt.expectedACMA, acma)
			}

			vary := rec.HeaderMap.Get("Vary")
            if vary != tt.expectedVary {
				t.Errorf("Expected VaryAge %s, got: %s", tt.expectedVary, vary)
			}

			// currently not working
			// cl := rec.HeaderMap.Get("Content-Length")
            // if cl != tt.expectedContentLength {
				// t.Errorf("Expected Content-Length %s, got: %s", tt.expectedContentLength, cl)
			// }

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
