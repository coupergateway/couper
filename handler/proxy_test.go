package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"go.avenga.cloud/couper/gateway/eval"
)

func TestProxy_ServeHTTP(t *testing.T) {
	type fields struct {
		evalContext *hcl.EvalContext
		log         *logrus.Entry
		options     *ProxyOptions
	}
	type args struct {
		rw  http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProxy(tt.fields.options, tt.fields.log, tt.fields.evalContext)
			if err != nil {
				t.Fatal(err)
			}
			// TODO: eval changes
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, tt.args.req)
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
		{"proxy url settings w/wildcard ctx", fields{eval.NewENVContext(nil), log.WithContext(nil), &ProxyOptions{Origin: "http://1.2.3.4", Hostname: "couper.io", Path: "/**", Context: emptyOptions}}, defaultReq.WithContext(context.WithValue(defaultReq.Context(), "route_wildcard", "/hans")), httptest.NewRequest("GET", "http://couper.io/hans", nil)},
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

	type testCase struct{
		name    string
		fields  fields
		args    args
		wantErr bool
	}

	tests := []testCase {
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProxy(tt.fields.options, tt.fields.log, tt.fields.evalContext)
			if err != nil {
				t.Fatal(err)
			}
			proxy := p.(*Proxy)
			if err := proxy.modifyResponse(tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("modifyResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
