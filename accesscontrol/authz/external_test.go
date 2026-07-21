package authz_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coupergateway/couper/accesscontrol/authz"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func respondStatus(status int) roundTripperFunc {
	return func(_ *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.WriteHeader(status)
		return rec.Result(), nil
	}
}

func TestExternal_Validate_Status(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		expKind string
	}{
		{"allow on 200", http.StatusOK, ""},
		{"invalid credentials on 401", http.StatusUnauthorized, "external_authz_invalid_credentials"},
		{"insufficient permissions on 403", http.StatusForbidden, "external_authz_insufficient_permissions"},
		{"deny on unexpected status", http.StatusBadGateway, "external_authz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			external := authz.NewExternal("test_ac", "http://authz.service/check", false, "", respondStatus(tc.status))

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			err := external.Validate(req)

			if tc.expKind == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			cErr, ok := err.(*errors.Error)
			if !ok {
				t.Fatalf("expected *errors.Error, got: %T", err)
			}

			kinds := cErr.Kinds()
			if len(kinds) == 0 || kinds[0] != tc.expKind {
				t.Errorf("expected most specific error kind %q, got: %v", tc.expKind, kinds)
			}
		})
	}
}

func TestExternal_Validate_CalloutRequest(t *testing.T) {
	var calloutReq *http.Request
	var calloutBody []byte

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calloutReq = req
		calloutBody, _ = io.ReadAll(req.Body)
		return respondStatus(http.StatusOK)(req)
	})

	external := authz.NewExternal("test_ac", "http://authz.service/check", false, "", transport)

	req := httptest.NewRequest(http.MethodDelete, "http://client.request/protected?a=b", nil)
	req.Header.Set("Authorization", "Bearer my-token")

	if err := external.Validate(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if calloutReq.Method != http.MethodPost {
		t.Errorf("expected POST callout, got: %s", calloutReq.Method)
	}
	if calloutReq.URL.String() != "http://authz.service/check" {
		t.Errorf("unexpected callout url: %s", calloutReq.URL)
	}
	if ct := calloutReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("unexpected content type: %q", ct)
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(calloutBody, &sent); err != nil {
		t.Fatalf("callout body is no valid json: %v", err)
	}

	clientRequest, _ := sent["client_request"].(map[string]interface{})
	if clientRequest == nil {
		t.Fatal("missing client_request object")
	}
	if clientRequest["method"] != http.MethodDelete {
		t.Errorf("expected serialized method DELETE, got: %v", clientRequest["method"])
	}
	if url, _ := clientRequest["url"].(string); !strings.HasSuffix(url, "/protected?a=b") {
		t.Errorf("unexpected serialized url: %v", clientRequest["url"])
	}
	headers, _ := clientRequest["headers"].(map[string]interface{})
	if headers == nil || headers["Authorization"] == nil {
		t.Errorf("expected serialized authorization header, got: %v", clientRequest["headers"])
	}
}

func TestExternal_Validate_ContextPropagation(t *testing.T) {
	respondBody := func(contentType, body string) roundTripperFunc {
		return func(_ *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			if contentType != "" {
				rec.Header().Set("Content-Type", contentType)
			}
			rec.WriteHeader(http.StatusOK)
			_, _ = rec.WriteString(body)
			return rec.Result(), nil
		}
	}

	contextData := func(req *http.Request) map[string]interface{} {
		acMap, _ := req.Context().Value(request.AccessControls).(map[string]interface{})
		data, _ := acMap["test_ac"].(map[string]interface{})
		return data
	}

	t.Run("json object response lands in access control context", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"sub":"clark.kent","roles":["reporter"]}`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data := contextData(req)
		if data == nil {
			t.Fatal("missing access control context data")
		}
		if data["sub"] != "clark.kent" {
			t.Errorf("unexpected sub: %v", data["sub"])
		}
		roles, _ := data["roles"].([]interface{})
		if len(roles) != 1 || roles[0] != "reporter" {
			t.Errorf("unexpected roles: %v", data["roles"])
		}
	})

	t.Run("response headers are exposed under headers", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				rec := httptest.NewRecorder()
				rec.Header().Set("Content-Type", "application/json")
				rec.Header().Set("X-Resolved-Identity", "clark.kent")
				rec.WriteHeader(http.StatusOK)
				_, _ = rec.WriteString(`{"sub":"clark.kent"}`)
				return rec.Result(), nil
			}))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		headers, _ := contextData(req)["headers"].(map[string]interface{})
		if headers["x-resolved-identity"] != "clark.kent" {
			t.Errorf("expected lower-cased x-resolved-identity, got: %v", headers)
		}
	})

	t.Run("invalid json fails closed", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"sub":`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		err := external.Validate(req)

		cErr, ok := err.(*errors.Error)
		if !ok {
			t.Fatalf("expected *errors.Error, got: %T", err)
		}
		if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
			t.Errorf("expected error kind external_authz, got: %v", kinds)
		}
	})

	t.Run("non-object json fails closed", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `[1,2]`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Error("expected an error for a non-object json response")
		}
	})

	t.Run("empty body still exposes headers", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", ""))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if _, ok := contextData(req)["headers"]; !ok {
			t.Error("expected headers in context data")
		}
	})

	t.Run("non-json response exposes headers without body properties", func(t *testing.T) {
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("text/plain", "OK"))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		data := contextData(req)
		if _, ok := data["headers"]; !ok {
			t.Error("expected headers in context data")
		}
		if data["sub"] != nil {
			t.Errorf("expected no body properties, got: %v", data)
		}
	})
}

func TestExternal_Validate_PermissionsProperty(t *testing.T) {
	respondJSON := func(body string) roundTripperFunc {
		return func(_ *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json")
			rec.WriteHeader(http.StatusOK)
			_, _ = rec.WriteString(body)
			return rec.Result(), nil
		}
	}

	granted := func(req *http.Request) []string {
		permissions, _ := req.Context().Value(request.GrantedPermissions).([]string)
		return permissions
	}

	newExternal := func(body string) *authz.External {
		return authz.NewExternal("test_ac", "http://authz.service/check", false, "perms", respondJSON(body))
	}

	t.Run("list property grants permissions", func(t *testing.T) {
		external := newExternal(`{"perms":["read","write"]}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 2 || p[0] != "read" || p[1] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("space-separated string grants permissions", func(t *testing.T) {
		external := newExternal(`{"perms":"read write"}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 2 || p[0] != "read" || p[1] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("appends to and dedupes against already granted permissions", func(t *testing.T) {
		external := newExternal(`{"perms":["read","write"]}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		ctx := context.WithValue(req.Context(), request.GrantedPermissions, []string{"admin", "read"})
		req = req.WithContext(ctx)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 3 || p[0] != "admin" || p[1] != "read" || p[2] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("missing property denies", func(t *testing.T) {
		external := newExternal(`{"sub":"clark.kent"}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		err := external.Validate(req)
		if err == nil {
			t.Fatal("expected an error for the missing permissions property")
		}
		logErr, ok := err.(errors.GoError)
		if !ok || !strings.Contains(logErr.LogError(), "missing perms permissions property") {
			t.Errorf("unexpected error: %v", err)
		}
		if p := granted(req); len(p) != 0 {
			t.Errorf("expected no granted permissions, got: %v", p)
		}
	})

	t.Run("invalid property type fails closed", func(t *testing.T) {
		for _, body := range []string{`{"perms":42}`, `{"perms":["read",42]}`} {
			external := newExternal(body)
			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

			err := external.Validate(req)
			cErr, ok := err.(*errors.Error)
			if !ok {
				t.Fatalf("expected *errors.Error for %s, got: %T", body, err)
			}
			if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
				t.Errorf("expected error kind external_authz for %s, got: %v", body, kinds)
			}
		}
	})
}

func TestExternal_Validate_IncludeTLS(t *testing.T) {
	newTLSRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "https://client.request/protected", nil)
		req.TLS = &tls.ConnectionState{
			Version:     tls.VersionTLS13,
			CipherSuite: tls.TLS_AES_128_GCM_SHA256,
			ServerName:  "client.request",
			PeerCertificates: []*x509.Certificate{{
				Subject:   pkix.Name{CommonName: "my-client"},
				Issuer:    pkix.Name{CommonName: "my-ca"},
				NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				NotAfter:  time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			}},
		}
		return req
	}

	captureBody := func(target *[]byte) roundTripperFunc {
		return func(req *http.Request) (*http.Response, error) {
			*target, _ = io.ReadAll(req.Body)
			return respondStatus(http.StatusOK)(req)
		}
	}

	t.Run("enabled", func(t *testing.T) {
		var calloutBody []byte
		external := authz.NewExternal("test_ac", "http://authz.service/check", true, "", captureBody(&calloutBody))

		if err := external.Validate(newTLSRequest()); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var sent map[string]interface{}
		if err := json.Unmarshal(calloutBody, &sent); err != nil {
			t.Fatalf("callout body is no valid json: %v", err)
		}

		metaTLS, _ := sent["metadata_tls"].(map[string]interface{})
		if metaTLS == nil {
			t.Fatal("missing metadata_tls object")
		}
		if metaTLS["version"] != "TLS 1.3" {
			t.Errorf("unexpected tls version: %v", metaTLS["version"])
		}
		if metaTLS["cipher_suite"] != "TLS_AES_128_GCM_SHA256" {
			t.Errorf("unexpected cipher suite: %v", metaTLS["cipher_suite"])
		}
		if metaTLS["server_name"] != "client.request" {
			t.Errorf("unexpected server name: %v", metaTLS["server_name"])
		}
		cert, _ := metaTLS["client_certificate"].(map[string]interface{})
		if cert == nil {
			t.Fatal("missing client_certificate object")
		}
		if cert["subject"] != "CN=my-client" {
			t.Errorf("unexpected certificate subject: %v", cert["subject"])
		}
		if cert["issuer"] != "CN=my-ca" {
			t.Errorf("unexpected certificate issuer: %v", cert["issuer"])
		}
	})

	t.Run("enabled without tls connection", func(t *testing.T) {
		var calloutBody []byte
		external := authz.NewExternal("test_ac", "http://authz.service/check", true, "", captureBody(&calloutBody))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var sent map[string]interface{}
		if err := json.Unmarshal(calloutBody, &sent); err != nil {
			t.Fatalf("callout body is no valid json: %v", err)
		}
		if _, exist := sent["metadata_tls"]; exist {
			t.Error("expected no metadata_tls object for non-tls connection")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		var calloutBody []byte
		external := authz.NewExternal("test_ac", "http://authz.service/check", false, "", captureBody(&calloutBody))

		if err := external.Validate(newTLSRequest()); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var sent map[string]interface{}
		if err := json.Unmarshal(calloutBody, &sent); err != nil {
			t.Fatalf("callout body is no valid json: %v", err)
		}
		if _, exist := sent["metadata_tls"]; exist {
			t.Error("expected no metadata_tls object when include_tls is disabled")
		}
	})
}

func TestExternal_Validate_TransportError(t *testing.T) {
	external := authz.NewExternal("test_ac", "http://authz.service/check", false, "", roundTripperFunc(
		func(_ *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}))

	req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
	err := external.Validate(req)

	cErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error, got: %T", err)
	}
	if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
		t.Errorf("expected error kind external_authz, got: %v", kinds)
	}
}

func TestExternal_Validate_EmptyURL(t *testing.T) {
	var calloutURL string
	external := authz.NewExternal("test_ac", "", false, "", roundTripperFunc(
		func(req *http.Request) (*http.Response, error) {
			calloutURL = req.URL.String()
			return respondStatus(http.StatusOK)(req)
		}))

	req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
	if err := external.Validate(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calloutURL != "/" {
		t.Errorf("expected callout path %q for backend-provided origin, got: %q", "/", calloutURL)
	}
}
