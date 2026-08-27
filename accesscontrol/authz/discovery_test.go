package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/accesscontrol/authz"
	"github.com/coupergateway/couper/config"
)

const wellKnownPath = "/.well-known/authzen-configuration"

// newDecisionPoint serves the AuthZEN configuration document plus an evaluation endpoint.
// metadata receives the origin of the server, so a test can claim a foreign one.
func newDecisionPoint(metadata func(origin string) string) (*httptest.Server, *[]string) {
	var mu sync.Mutex
	var calloutPaths []string

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.HandleFunc(wellKnownPath, func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(metadata(server.URL)))
	})
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		mu.Lock()
		calloutPaths = append(calloutPaths, req.URL.Path)
		mu.Unlock()

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"decision": true}`))
	})

	return server, &calloutPaths
}

func newDiscoveringExternal(t *testing.T, conf *config.ExternalAuthZ) (*authz.External, error) {
	t.Helper()

	conf.Remain = &hclsyntax.Body{}

	return authz.NewExternal(context.Background(), conf, http.DefaultTransport, newTestLogEntry())
}

func TestExternal_Discovery(t *testing.T) {
	t.Run("the callout goes to the advertised endpoint", func(t *testing.T) {
		server, calloutPaths := newDecisionPoint(func(origin string) string {
			return `{"policy_decision_point": "` + origin +
				`", "access_evaluation_endpoint": "` + origin + `/custom/evaluate"}`
		})
		defer server.Close()

		external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
			ConfigurationURL: server.URL + wellKnownPath,
			Name:             "test_ac",
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err = external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(*calloutPaths) != 1 || (*calloutPaths)[0] != "/custom/evaluate" {
			t.Errorf("unexpected callout paths: %v", *calloutPaths)
		}
	})

	t.Run("a batch configuration uses the evaluations endpoint", func(t *testing.T) {
		server, calloutPaths := newDecisionPoint(func(origin string) string {
			return `{"policy_decision_point": "` + origin +
				`", "access_evaluation_endpoint": "` + origin + `/one",` +
				`"access_evaluations_endpoint": "` + origin + `/many"}`
		})
		defer server.Close()

		external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
			ConfigurationURL:    server.URL + wellKnownPath,
			EvaluatePermissions: []string{"can_read"},
			Name:                "test_ac",
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		// the stub answers a single decision, which is one evaluation short
		_ = external.Validate(req)

		if len(*calloutPaths) != 1 || (*calloutPaths)[0] != "/many" {
			t.Errorf("unexpected callout paths: %v", *calloutPaths)
		}
	})

	// OpenFGA roots its identifiers under /stores: the origin and the tenant of the
	// configuration URL stay pinned, only the path prefix of the identifier is free.
	t.Run("a store-scoped identifier of a tenant configuration is accepted", func(t *testing.T) {
		const tenantWellKnown = wellKnownPath + "/01ARZ"

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)
		defer server.Close()

		mux.HandleFunc(tenantWellKnown, func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write([]byte(`{"policy_decision_point": "` + server.URL + `/stores/01ARZ",` +
				`"access_evaluation_endpoint": "` + server.URL + `/stores/01ARZ/access/v1/evaluation"}`))
		})
		mux.HandleFunc("/stores/01ARZ/access/v1/evaluation", func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write([]byte(`{"decision": true}`))
		})

		external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
			ConfigurationURL: server.URL + tenantWellKnown,
			Name:             "test_ac",
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err = external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("a store-scoped identifier must keep the origin and the tenant", func(t *testing.T) {
		const tenantWellKnown = wellKnownPath + "/01ARZ"

		for _, tc := range []struct {
			name       string
			identifier func(origin string) string
		}{
			{"foreign origin", func(string) string { return "https://evil.example/stores/01ARZ" }},
			{"foreign tenant", func(origin string) string { return origin + "/stores/OTHER" }},
			{"tenant inside a segment", func(origin string) string { return origin + "/stores01ARZ" }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mux := http.NewServeMux()
				server := httptest.NewServer(mux)
				defer server.Close()

				mux.HandleFunc(tenantWellKnown, func(rw http.ResponseWriter, _ *http.Request) {
					rw.Header().Set("Content-Type", "application/json")
					_, _ = rw.Write([]byte(`{"policy_decision_point": "` + tc.identifier(server.URL) + `",` +
						`"access_evaluation_endpoint": "` + server.URL + `/evaluate"}`))
				})

				external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
					ConfigurationURL: server.URL + tenantWellKnown,
					Name:             "test_ac",
				})
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}

				req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
				err = external.Validate(req)
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.(interface{ LogError() string }).LogError(), "does not match") {
					t.Errorf("expected an identifier mismatch, got: %v", err)
				}
			})
		}
	})

	for _, tc := range []struct {
		name     string
		metadata func(origin string) string
		expError string
	}{
		{
			// Without a tenant the identifier stays strict: an own path prefix is only
			// acceptable when a tenant pins the suffix.
			"a same-origin identifier under another path is rejected without a tenant",
			func(origin string) string {
				return `{"policy_decision_point": "` + origin + `/stores",` +
					`"access_evaluation_endpoint": "` + origin + `/evaluate"}`
			},
			"does not match",
		},
		{
			"a foreign policy decision point is rejected",
			func(origin string) string {
				return `{"policy_decision_point": "https://evil.example",` +
					`"access_evaluation_endpoint": "` + origin + `/evaluate"}`
			},
			"does not match",
		},
		{
			// The backend pins scheme and host, so a foreign endpoint would be sent to the
			// configured origin without any diagnosis.
			"an endpoint on a foreign origin is rejected",
			func(origin string) string {
				return `{"policy_decision_point": "` + origin +
					`", "access_evaluation_endpoint": "https://evil.example/evaluate"}`
			},
			"not on the origin",
		},
		{
			"a protocol-relative endpoint is rejected",
			func(origin string) string {
				return `{"policy_decision_point": "` + origin +
					`", "access_evaluation_endpoint": "//evil.example/evaluate"}`
			},
			"not on the origin",
		},
		{
			"a missing evaluation endpoint is rejected",
			func(origin string) string {
				return `{"policy_decision_point": "` + origin + `"}`
			},
			"missing access_evaluation_endpoint",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calloutPaths := newDecisionPoint(tc.metadata)
			defer server.Close()

			external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
				ConfigurationURL: server.URL + wellKnownPath,
				Name:             "test_ac",
			})
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			err = external.Validate(req)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.(interface{ LogError() string }).LogError(), tc.expError) {
				t.Errorf("expected an error containing %q, got: %v", tc.expError, err)
			}
			if len(*calloutPaths) != 0 {
				t.Errorf("expected no callout, got: %v", *calloutPaths)
			}
		})
	}

	t.Run("invalid configuration urls are configuration errors", func(t *testing.T) {
		for _, confURL := range []string{
			"https://pdp.example.com/metadata",
			"https://pdp.example.com/.well-known/authzen-configuration-backup",
			"/.well-known/authzen-configuration", // relative: no origin to check against
		} {
			_, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
				ConfigurationURL: confURL,
				Name:             "test_ac",
			})
			if err == nil {
				t.Errorf("expected an error for %q", confURL)
			}
		}
	})

	t.Run("a stale document serves callouts while the refresh fails", func(t *testing.T) {
		var mu sync.Mutex
		served := false

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)
		defer server.Close()

		mux.HandleFunc(wellKnownPath, func(rw http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			if served {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			served = true
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write([]byte(`{"policy_decision_point": "` + server.URL +
				`", "access_evaluation_endpoint": "` + server.URL + `/evaluate"}`))
		})
		mux.HandleFunc("/evaluate", func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write([]byte(`{"decision": true}`))
		})

		external, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
			ConfigurationMaxStale: "1h",
			ConfigurationTTL:      "50ms",
			ConfigurationURL:      server.URL + wellKnownPath,
			Name:                  "test_ac",
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err = external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		time.Sleep(500 * time.Millisecond) // let the refresh fail at least once

		req = httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err = external.Validate(req); err != nil {
			t.Fatalf("expected the stale document to serve the callout, got: %v", err)
		}
	})
}
