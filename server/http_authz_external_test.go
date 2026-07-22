package server_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/coupergateway/couper/internal/test"
)

func TestAuthzExternal_Callout(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/authz_external/01_couper.hcl", helper)
	defer shutdown()

	for _, tc := range []struct {
		name          string
		authorization string
		expStatus     int
		expErrorType  string
	}{
		{"valid credentials", "Bearer valid", http.StatusNoContent, ""},
		{"missing credentials", "", http.StatusUnauthorized, "authz_external_invalid_credentials"},
		{"insufficient permissions", "Bearer forbidden", http.StatusForbidden, "authz_external_insufficient_permissions"},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
			helper.Must(err)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}

			res, err := client.Do(req)
			helper.Must(err)
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()

			if res.StatusCode != tc.expStatus {
				st.Errorf("expected status %d, got: %d", tc.expStatus, res.StatusCode)
			}

			if tc.expErrorType == "" {
				return
			}

			var loggedType string
			for _, entry := range hook.AllEntries() {
				if errorType, ok := entry.Data["error_type"].(string); ok && entry.Data["port"] == "8080" {
					loggedType = errorType
				}
			}
			if loggedType != tc.expErrorType {
				st.Errorf("expected logged error_type %q, got: %q", tc.expErrorType, loggedType)
			}
		})
	}
}

func TestAuthzExternal_ErrorHandler(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/authz_external/02_couper.hcl", helper)
	defer shutdown()
	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
	}

	expChallenge := `Bearer resource_metadata="http://protected.example/.well-known/oauth-protected-resource"`
	if challenge := res.Header.Get("Www-Authenticate"); challenge != expChallenge {
		t.Errorf("expected challenge %q, got: %q", expChallenge, challenge)
	}
}
