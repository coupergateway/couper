package handler_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avenga/couper/config/body"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
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

	log, hook := logrustest.NewNullLogger()

	beConf := &config.Backend{
		Remain: body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
			"origin": &hcl.Attribute{
				Name: "origin",
				Expr: hcltest.MockExprLiteral(cty.StringVal(origin.URL)),
			},
		}}),
		OpenAPI: []*config.OpenAPI{{
			File: filepath.Join("testdata/validation/backend_01_openapi.yaml"),
		}},
		RequestBodyLimit: "64MiB",
	}

	proxyOpts, err := handler.NewProxyOptions(beConf, &handler.CORSOptions{}, config.DefaultSettings.NoProxyFromEnv, errors.DefaultJSON, "api", nil, nil)
	helper.Must(err)

	backend, err := handler.NewProxy(proxyOpts, log.WithContext(context.Background()), nil, eval.NewENVContext(nil))
	helper.Must(err)

	tests := []struct {
		name, method, path string
		body               io.Reader
		wantBody           bool
		wantErrLog         string
	}{
		{"GET without required query", http.MethodGet, "/a?b", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with required query", http.MethodGet, "/a?b=value", nil, false, ""},
		{"GET with required path", http.MethodGet, "/a/value", nil, false, ""},
		{"GET with required path missing", http.MethodGet, "/a//", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with optional query", http.MethodGet, "/b", nil, false, ""},
		{"GET with optional path param", http.MethodGet, "/b/a", nil, false, ""},
		{"GET with required json body", http.MethodGet, "/json", strings.NewReader(`["hans", "wurst"]`), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			rec := httptest.NewRecorder()

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			hook.Reset()
			backend.ServeHTTP(rec, req)
			rec.Flush()

			res := rec.Result()

			if tt.wantErrLog == "" && res.StatusCode != http.StatusOK {
				t.Errorf("Expected OK, got: %s", res.Status)
			}

			if tt.wantErrLog != "" {
				var found bool
				for _, entry := range hook.Entries {
					if entry.Message == tt.wantErrLog {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error log: %q", tt.wantErrLog)
				}
			}

			if tt.wantBody {
				n, err := io.Copy(ioutil.Discard, res.Body)
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
