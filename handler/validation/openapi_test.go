package validation_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/test"
)

// TestOpenAPIValidator_ValidateRequest should not test the openapi validation functionality but must
// ensure that all required parameters (query, body) are set as required and bodies are still readable.
func TestOpenAPIValidator_ValidateRequest(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ct := req.Header.Get("Content-Type")
		if ct != "" {
			n, err := io.Copy(ioutil.Discard, req.Body)
			helper.Must(err)
			if n == 0 {
				t.Error("Expected body content")
			}
		}
		if req.Header.Get("Content-Type") == "application/json" {
			rw.Header().Set("Content-Type", "application/json")
			_, err := rw.Write([]byte(`{"id": 123, "name": "hans"}`))
			helper.Must(err)
		}
	}))

	log, hook := test.NewLogger()
	logger := log.WithContext(context.Background())
	beConf := &config.Backend{
		Remain: body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
			"origin": &hcl.Attribute{
				Name: "origin",
				Expr: hcltest.MockExprLiteral(cty.StringVal(origin.URL)),
			},
		}}),
		OpenAPI: &config.OpenAPI{
			File: filepath.Join("testdata/backend_01_openapi.yaml"),
		},
	}
	openAPI, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	helper.Must(err)

	backend := transport.NewBackend(beConf.Remain, &transport.Config{}, &transport.BackendOptions{
		OpenAPI: openAPI,
	}, logger)

	tests := []struct {
		name, path string
		body       io.Reader
		wantBody   bool
		wantErrLog string
	}{
		{"GET without required query", "/a?b", nil, false, "backend validation error: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with required query", "/a?b=value", nil, false, ""},
		{"GET with required path", "/a/value", nil, false, ""},
		{"GET with required path missing", "/a//", nil, false, "backend validation error: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with optional query", "/b", nil, false, ""},
		{"GET with optional path param", "/b/a", nil, false, ""},
		{"GET with required json body", "/json", strings.NewReader(`["hans", "wurst"]`), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, tt.body)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			hook.Reset()
			var res *http.Response
			res, err = backend.RoundTrip(req)
			if err != nil && tt.wantErrLog == "" {
				t.Error(err)
				return
			}

			if tt.wantErrLog != "" {
				entry := hook.LastEntry()
				if entry.Message != tt.wantErrLog {
					t.Errorf("Expected error log:\nwant:\t%q\ngot:\t%s", tt.wantErrLog, entry.Message)
				}
			}

			if tt.wantBody {
				var n int64
				n, err = io.Copy(ioutil.Discard, res.Body)
				if err != nil {
					t.Error(err)
				}
				if n == 0 {
					t.Error("Expected a response body")
				}
			}

			if t.Failed() {
				for _, entry := range hook.Entries {
					t.Log(entry.String())
				}
			}
		})
	}
}

func TestOpenAPIValidator_RelativeServerURL(t *testing.T) {
	helper := test.New(t)

	log, hook := test.NewLogger()
	logger := log.WithContext(context.Background())
	beConf := &config.Backend{
		Remain: body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
			"origin": &hcl.Attribute{
				Name: "origin",
				Expr: hcltest.MockExprLiteral(cty.StringVal("https://httpbin.org")),
			},
		}}),
		OpenAPI: &config.OpenAPI{
			File: filepath.Join("testdata/backend_02_openapi.yaml"),
		},
	}
	openAPI, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	helper.Must(err)

	backend := transport.NewBackend(beConf.Remain, &transport.Config{}, &transport.BackendOptions{
		OpenAPI: openAPI,
	}, logger)

	req := httptest.NewRequest(http.MethodGet, "https://httpbin.org/anything", nil)

	hook.Reset()
	_, err = backend.RoundTrip(req)
	if err != nil {
		helper.Must(err)
	}

	if t.Failed() {
		for _, entry := range hook.Entries {
			t.Log(entry.String())
		}
	}
}

func TestOpenAPIValidator_TemplateVariables(t *testing.T) {
	helper := test.New(t)

	origin := test.NewBackend()
	defer origin.Close()

	log, hook := test.NewLogger()
	logger := log.WithContext(context.Background())

	type testCase struct {
		name, origin, hostname string
	}

	openAPI, err := validation.NewOpenAPIOptions(&config.OpenAPI{
		File: filepath.Join("testdata/backend_04_openapi.yaml"),
	})
	helper.Must(err)

	for _, tc := range []testCase{
		{name: "tpl url", origin: "https://api.example.com"},
		{name: "relative url", origin: origin.Addr()},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			beConf := &config.Backend{
				Remain: body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
					"origin": &hcl.Attribute{
						Name: "origin",
						Expr: hcltest.MockExprLiteral(cty.StringVal(tc.origin)),
					},
					"hostname": &hcl.Attribute{
						Name: "hostname",
						Expr: hcltest.MockExprLiteral(cty.StringVal(tc.hostname)),
					},
					"proxy": &hcl.Attribute{
						Name: "proxy",
						Expr: hcltest.MockExprLiteral(cty.StringVal(origin.Addr())),
					},
					"path": &hcl.Attribute{
						Name: "path",
						Expr: hcltest.MockExprLiteral(cty.StringVal("/anything")),
					},
				}}),
			}

			backend := transport.NewBackend(beConf.Remain, &transport.Config{}, &transport.BackendOptions{
				OpenAPI: openAPI,
			}, logger)

			req := httptest.NewRequest(http.MethodGet, "https://test.local/", nil)

			hook.Reset()
			_, err = backend.RoundTrip(req)
			if err != nil && err.Error() != "backend error" {
				subT.Error(err)
			}

			if subT.Failed() {
				for _, entry := range hook.Entries {
					subT.Log(entry.String())
				}
			}
		})
	}
}

func TestOpenAPIValidator_NonCanonicalServerURL(t *testing.T) {
	helper := test.New(t)

	origin := test.NewBackend()
	defer origin.Close()

	log, hook := test.NewLogger()
	logger := log.WithContext(context.Background())

	openAPI, err := validation.NewOpenAPIOptions(&config.OpenAPI{
		File: filepath.Join("testdata/backend_03_openapi.yaml"),
	})
	helper.Must(err)

	tests := []struct {
		name, url string
	}{
		{"http", "http://api.example.com"},
		{"http80", "http://api.example.com:80"},
		{"http443", "http://api.example.com:443"},
		{"https", "https://api.example.com"},
		{"https443", "https://api.example.com:443"},
		{"https80", "https://api.example.com:80"},
		{"httpsHigh", "https://api.example.com:12345"},
		{"httpHigh", "http://api.example.com:12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			Remain := body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
				"origin": &hcl.Attribute{
					Name: "origin",
					Expr: hcltest.MockExprLiteral(cty.StringVal(tt.url)),
				},
				"path": &hcl.Attribute{
					Name: "path",
					Expr: hcltest.MockExprLiteral(cty.StringVal("/anything")),
				},
				"proxy": &hcl.Attribute{
					Name: "proxy",
					Expr: hcltest.MockExprLiteral(cty.StringVal(origin.Addr())),
				},
			}})

			backend := transport.NewBackend(Remain, &transport.Config{}, &transport.BackendOptions{
				OpenAPI: openAPI,
			}, logger)
			hook.Reset()

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			_, err = backend.RoundTrip(req)
			if err != nil && err.Error() != "backend error" {
				subT.Error(err)
			}

			if subT.Failed() {
				for _, entry := range hook.Entries {
					subT.Log(entry.String())
				}
			}
		})
	}
}
