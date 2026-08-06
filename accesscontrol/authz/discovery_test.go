package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

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

	for _, tc := range []struct {
		name     string
		metadata func(origin string) string
		expError string
	}{
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

	t.Run("a configuration url without the well-known path is a configuration error", func(t *testing.T) {
		_, err := newDiscoveringExternal(t, &config.ExternalAuthZ{
			ConfigurationURL: "https://pdp.example.com/metadata",
			Name:             "test_ac",
		})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
}
