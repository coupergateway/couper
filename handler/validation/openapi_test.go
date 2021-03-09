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
	logrustest "github.com/sirupsen/logrus/hooks/test"
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

	log, hook := logrustest.NewNullLogger()
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

	backend := transport.NewBackend(beConf.Remain, &transport.Config{}, logger, openAPI)

	tests := []struct {
		name, path string
		body       io.Reader
		wantBody   bool
		wantErrLog string
	}{
		{"GET without required query", "/a?b", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with required query", "/a?b=value", nil, false, ""},
		{"GET with required path", "/a/value", nil, false, ""},
		{"GET with required path missing", "/a//", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
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
			res, err := backend.RoundTrip(req)
			if err != nil && tt.wantErrLog == "" {
				helper.Must(err)
			}

			if tt.wantErrLog != "" {
				var found bool
				for _, entry := range hook.Entries {
					if valEntry, ok := entry.Data["validation"]; ok {
						if list, ok := valEntry.([]string); ok {
							for _, valMsg := range list {
								if valMsg == tt.wantErrLog {
									found = true
									break
								}
							}
						}
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
