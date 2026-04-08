package mcp

import (
	"encoding/json"
	"net/url"
	"testing"
)

func newTestRewriter(t *testing.T) *OAuthRewriter {
	t.Helper()
	r, err := NewOAuthRewriter("https://gateway.example.com", "https://mcp.example.com")
	if err != nil {
		t.Fatalf("NewOAuthRewriter: %v", err)
	}
	return r
}

// ── constructor ──────────────────────────────────────────────────────────────

func TestNewOAuthRewriter_StripsTrailingSlash(t *testing.T) {
	r, err := NewOAuthRewriter("https://proxy.example.com/", "https://upstream.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ProxyBase() != "https://proxy.example.com" {
		t.Errorf("ProxyBase() = %q, want %q", r.ProxyBase(), "https://proxy.example.com")
	}
	if r.UpstreamOrigin() != "https://upstream.example.com" {
		t.Errorf("UpstreamOrigin() = %q, want %q", r.UpstreamOrigin(), "https://upstream.example.com")
	}
}

func TestNewOAuthRewriter_RejectsEmptyProxyBase(t *testing.T) {
	_, err := NewOAuthRewriter("", "https://mcp.example.com")
	if err == nil {
		t.Error("expected error for empty proxyBase")
	}
}

func TestNewOAuthRewriter_RejectsEmptyUpstream(t *testing.T) {
	_, err := NewOAuthRewriter("https://proxy.example.com", "")
	if err == nil {
		t.Error("expected error for empty upstreamOrigin")
	}
}

func TestNewOAuthRewriter_RejectsInvalidURL(t *testing.T) {
	_, err := NewOAuthRewriter("not a url", "https://mcp.example.com")
	if err == nil {
		t.Error("expected error for invalid proxyBase URL")
	}
}

// ── ProtectedResourceMetadata rewriting ─────────────────────────────────────

func TestRewriteProtectedResourceMetadata_RewritesResourceAndAuthServers(t *testing.T) {
	r := newTestRewriter(t)

	upstream := `{
		"resource": "https://mcp.example.com",
		"authorization_servers": ["https://auth.mcp.example.com"],
		"bearer_methods_supported": ["header"]
	}`

	result, err := r.RewriteProtectedResourceMetadata([]byte(upstream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	var resource string
	if err := json.Unmarshal(out["resource"], &resource); err != nil {
		t.Fatalf("resource field: %v", err)
	}
	if resource != "https://gateway.example.com" {
		t.Errorf("resource = %q, want %q", resource, "https://gateway.example.com")
	}

	var authServers []string
	if err := json.Unmarshal(out["authorization_servers"], &authServers); err != nil {
		t.Fatalf("authorization_servers field: %v", err)
	}
	if len(authServers) != 1 || authServers[0] != "https://gateway.example.com" {
		t.Errorf("authorization_servers = %v, want [%q]", authServers, "https://gateway.example.com")
	}

	// Unknown fields must be preserved.
	if _, ok := out["bearer_methods_supported"]; !ok {
		t.Error("bearer_methods_supported should be preserved")
	}
}

func TestRewriteProtectedResourceMetadata_InvalidJSON(t *testing.T) {
	r := newTestRewriter(t)
	_, err := r.RewriteProtectedResourceMetadata([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

// ── AuthorizationServerMetadata rewriting ────────────────────────────────────

func TestRewriteAuthorizationServerMetadata_RewritesIssuerAndEndpoints(t *testing.T) {
	r := newTestRewriter(t)

	upstream := `{
		"issuer": "https://mcp.example.com",
		"authorization_endpoint": "https://mcp.example.com/oauth/authorize",
		"token_endpoint": "https://mcp.example.com/oauth/token",
		"registration_endpoint": "https://mcp.example.com/oauth/register",
		"scopes_supported": ["read","write"]
	}`

	result, err := r.RewriteAuthorizationServerMetadata([]byte(upstream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	checkStringField := func(field, want string) {
		t.Helper()
		var val string
		if err := json.Unmarshal(out[field], &val); err != nil {
			t.Fatalf("%s field: %v", field, err)
		}
		if val != want {
			t.Errorf("%s = %q, want %q", field, val, want)
		}
	}

	checkStringField("issuer", "https://gateway.example.com")
	checkStringField("token_endpoint", "https://gateway.example.com/token")
	checkStringField("registration_endpoint", "https://gateway.example.com/register")

	// authorization_endpoint is rewritten to proxy so the proxy can intercept
	// and rewrite the resource parameter before redirecting to upstream.
	checkStringField("authorization_endpoint", "https://gateway.example.com/authorize")

	// Unknown fields must be preserved.
	if _, ok := out["scopes_supported"]; !ok {
		t.Error("scopes_supported should be preserved")
	}
}

func TestRewriteAuthorizationServerMetadata_NoTokenEndpoint(t *testing.T) {
	r := newTestRewriter(t)

	// Upstream has no token_endpoint (unusual but should not panic).
	upstream := `{
		"issuer": "https://mcp.example.com",
		"authorization_endpoint": "https://mcp.example.com/oauth/authorize"
	}`

	result, err := r.RewriteAuthorizationServerMetadata([]byte(upstream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// token_endpoint and registration_endpoint should remain absent.
	if _, ok := out["token_endpoint"]; ok {
		t.Error("token_endpoint should not be added when absent in upstream")
	}
	if _, ok := out["registration_endpoint"]; ok {
		t.Error("registration_endpoint should not be added when absent in upstream")
	}
}

func TestRewriteAuthorizationServerMetadata_InvalidJSON(t *testing.T) {
	r := newTestRewriter(t)
	_, err := r.RewriteAuthorizationServerMetadata([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

// ── resource param rewriting (form-encoded) ──────────────────────────────────

func TestRewriteResourceParamToUpstream_FormEncoded_MatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte("grant_type=client_credentials&resource=https%3A%2F%2Fgateway.example.com&client_id=abc")
	result, err := r.RewriteResourceParamToUpstream(body, "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed := parseFormBody(t, result)

	if parsed["resource"] != "https://mcp.example.com" {
		t.Errorf("resource = %q, want %q", parsed["resource"], "https://mcp.example.com")
	}
	if parsed["grant_type"] != "client_credentials" {
		t.Error("other fields should be preserved")
	}
}

func TestRewriteResourceParamToUpstream_FormEncoded_NonMatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte("grant_type=client_credentials&resource=https%3A%2F%2Fother.example.com")
	result, err := r.RewriteResourceParamToUpstream(body, "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed := parseFormBody(t, result)

	if parsed["resource"] != "https://other.example.com" {
		t.Errorf("resource should be unchanged, got %q", parsed["resource"])
	}
}

func TestRewriteResourceParamToUpstream_FormEncoded_NoResourceParam(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte("grant_type=client_credentials&client_id=abc")
	result, err := r.RewriteResourceParamToUpstream(body, "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed := parseFormBody(t, result)

	if _, ok := parsed["resource"]; ok {
		t.Error("resource should not be added when absent")
	}
}

// ── resource param rewriting (JSON body) ─────────────────────────────────────

func TestRewriteResourceParamToUpstream_JSON_MatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte(`{"grant_type":"client_credentials","resource":"https://gateway.example.com","client_id":"abc"}`)
	result, err := r.RewriteResourceParamToUpstream(body, "application/json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]string
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	if out["resource"] != "https://mcp.example.com" {
		t.Errorf("resource = %q, want %q", out["resource"], "https://mcp.example.com")
	}
	if out["grant_type"] != "client_credentials" {
		t.Error("other fields should be preserved")
	}
}

func TestRewriteResourceParamToUpstream_JSON_NonMatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte(`{"resource":"https://other.example.com"}`)
	result, err := r.RewriteResourceParamToUpstream(body, "application/json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]string
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	if out["resource"] != "https://other.example.com" {
		t.Errorf("resource should be unchanged, got %q", out["resource"])
	}
}

func TestRewriteResourceParamToUpstream_UnknownContentType(t *testing.T) {
	r := newTestRewriter(t)

	body := []byte("raw body")
	result, err := r.RewriteResourceParamToUpstream(body, "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != "raw body" {
		t.Errorf("body should be returned unchanged, got %q", string(result))
	}
}

func TestRewriteResourceParamToUpstream_JSON_InvalidJSON(t *testing.T) {
	r := newTestRewriter(t)
	_, err := r.RewriteResourceParamToUpstream([]byte(`not json`), "application/json")
	if err == nil {
		t.Error("expected error for invalid JSON body")
	}
}

// ── resource param rewriting (query string) ──────────────────────────────────

func TestRewriteResourceParamInQuery_MatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	rawURL := "https://mcp.example.com/token?grant_type=client_credentials&resource=https%3A%2F%2Fgateway.example.com"
	result, err := r.RewriteResourceParamInQuery(rawURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := parseQueryURL(t, result)

	if q["resource"] != "https://mcp.example.com" {
		t.Errorf("resource = %q, want %q", q["resource"], "https://mcp.example.com")
	}
}

func TestRewriteResourceParamInQuery_NonMatchingResource(t *testing.T) {
	r := newTestRewriter(t)

	rawURL := "https://mcp.example.com/token?resource=https%3A%2F%2Fother.example.com"
	result, err := r.RewriteResourceParamInQuery(rawURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value should be unchanged; compare after round-tripping through url.Parse
	// to normalise encoding.
	u1, _ := url.Parse(rawURL)
	u2, _ := url.Parse(result)
	if u1.Query().Get("resource") != u2.Query().Get("resource") {
		t.Errorf("resource should be unchanged\n  got:  %q\n  want: %q",
			u2.Query().Get("resource"), u1.Query().Get("resource"))
	}
}

func TestRewriteResourceParamInQuery_NoResourceParam(t *testing.T) {
	r := newTestRewriter(t)

	rawURL := "https://mcp.example.com/token?grant_type=client_credentials"
	result, err := r.RewriteResourceParamInQuery(rawURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := parseQueryURL(t, result)

	if _, ok := q["resource"]; ok {
		t.Error("resource should not be added when absent")
	}
}

func TestRewriteResourceParamInQuery_InvalidURL(t *testing.T) {
	r := newTestRewriter(t)
	_, err := r.RewriteResourceParamInQuery("://bad url")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

func parseFormBody(t *testing.T, b []byte) map[string]string {
	t.Helper()
	vals, err := url.ParseQuery(string(b))
	if err != nil {
		t.Fatalf("parseFormBody: %v", err)
	}
	m := make(map[string]string, len(vals))
	for k, v := range vals {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func parseQueryURL(t *testing.T, rawURL string) map[string]string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parseQueryURL: %v", err)
	}
	m := make(map[string]string)
	for k, v := range u.Query() {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}
