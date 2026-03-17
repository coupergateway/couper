package mcp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// OAuthRewriter is a value object that encapsulates URL rewriting logic for
// MCP OAuth proxy flows. It works with plain data structures and has no
// dependency on http.Request or http.Response — those concerns live in
// OAuthHandler.
//
// The proxy sits between an MCP client and an upstream MCP server:
//
//	MCP client  →  Couper proxy (proxyBase)  →  upstream MCP server (upstreamOrigin)
//
// OAuth tokens are bound to the resource value used when they were issued.
// Without rewriting, a client that discovers resource=proxyBase would obtain a
// token bound to proxyBase, which the upstream server would reject because it
// expects tokens bound to its own origin.
//
// The rewriter solves this by:
//   - Advertising proxyBase as the resource in protected-resource metadata
//   - Advertising proxy endpoints for token and registration flows
//   - Translating resource=proxyBase back to upstreamOrigin when forwarding
//     token and registration requests to the upstream server
type OAuthRewriter struct {
	proxyBase      string // e.g. "https://gateway.example.com"
	upstreamOrigin string // e.g. "https://mcp.example.com"
}

// NewOAuthRewriter creates an OAuthRewriter. Both proxyBase and upstreamOrigin
// must be non-empty URL origins (scheme + host, no trailing slash).
func NewOAuthRewriter(proxyBase, upstreamOrigin string) (*OAuthRewriter, error) {
	if proxyBase == "" {
		return nil, fmt.Errorf("mcp oauth rewriter: proxyBase must not be empty")
	}
	if upstreamOrigin == "" {
		return nil, fmt.Errorf("mcp oauth rewriter: upstreamOrigin must not be empty")
	}

	proxyBase = strings.TrimRight(proxyBase, "/")
	upstreamOrigin = strings.TrimRight(upstreamOrigin, "/")

	if _, err := url.ParseRequestURI(proxyBase); err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: invalid proxyBase %q: %w", proxyBase, err)
	}
	if _, err := url.ParseRequestURI(upstreamOrigin); err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: invalid upstreamOrigin %q: %w", upstreamOrigin, err)
	}

	return &OAuthRewriter{
		proxyBase:      proxyBase,
		upstreamOrigin: upstreamOrigin,
	}, nil
}

// ProxyBase returns the proxy base URL (no trailing slash).
func (r *OAuthRewriter) ProxyBase() string { return r.proxyBase }

// UpstreamOrigin returns the upstream origin URL (no trailing slash).
func (r *OAuthRewriter) UpstreamOrigin() string { return r.upstreamOrigin }

// ProtectedResourceMetadata is the JSON structure returned by
// /.well-known/oauth-protected-resource (RFC 9728).
type ProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers,omitempty"`
	// Additional fields are preserved as-is when rewriting.
	Extra map[string]json.RawMessage `json:"-"`
}

// RewriteProtectedResourceMetadata takes the raw JSON body from the upstream
// /.well-known/oauth-protected-resource response and rewrites it so that:
//   - resource → proxyBase
//   - authorization_servers → [proxyBase]  (clients discover token endpoint via proxy)
//
// All other fields from the upstream response are preserved.
func (r *OAuthRewriter) RewriteProtectedResourceMetadata(upstreamBody []byte) ([]byte, error) {
	// Decode into a generic map to preserve unknown fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(upstreamBody, &raw); err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: parse protected-resource metadata: %w", err)
	}

	// Overwrite the fields we care about; keep everything else.
	resourceJSON, _ := json.Marshal(r.proxyBase)
	raw["resource"] = resourceJSON

	authServersJSON, _ := json.Marshal([]string{r.proxyBase})
	raw["authorization_servers"] = authServersJSON

	return json.Marshal(raw)
}

// AuthorizationServerMetadata is the JSON structure returned by
// /.well-known/oauth-authorization-server (RFC 8414).
type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint,omitempty"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
}

// RewriteAuthorizationServerMetadata takes the raw JSON body from the upstream
// /.well-known/oauth-authorization-server response and rewrites it so that:
//   - issuer → proxyBase
//   - token_endpoint → proxyBase + "/token"  (clients POST tokens via proxy)
//   - registration_endpoint → proxyBase + "/register"  (clients register via proxy)
//   - authorization_endpoint is intentionally left pointing at the upstream
//     because browser redirects cannot be proxied through the API gateway
//
// All other fields from the upstream response are preserved.
func (r *OAuthRewriter) RewriteAuthorizationServerMetadata(upstreamBody []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(upstreamBody, &raw); err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: parse authorization-server metadata: %w", err)
	}

	issuerJSON, _ := json.Marshal(r.proxyBase)
	raw["issuer"] = issuerJSON

	// Rewrite token_endpoint only if the upstream has one.
	if _, hasToken := raw["token_endpoint"]; hasToken {
		tokenEndpointJSON, _ := json.Marshal(r.proxyBase + "/token")
		raw["token_endpoint"] = tokenEndpointJSON
	}

	// Rewrite registration_endpoint only if the upstream has one.
	if _, hasReg := raw["registration_endpoint"]; hasReg {
		regEndpointJSON, _ := json.Marshal(r.proxyBase + "/register")
		raw["registration_endpoint"] = regEndpointJSON
	}

	// Rewrite authorization_endpoint so the proxy can intercept and rewrite
	// the resource parameter before redirecting to the upstream.
	if _, hasAuth := raw["authorization_endpoint"]; hasAuth {
		authEndpointJSON, _ := json.Marshal(r.proxyBase + "/authorize")
		raw["authorization_endpoint"] = authEndpointJSON
	}

	return json.Marshal(raw)
}

// RewriteResourceParamToUpstream rewrites the "resource" parameter value in a
// form-encoded or JSON body of a /token or /register request. When a client
// sends resource=proxyBase the upstream expects resource=upstreamOrigin.
//
// The body format is detected automatically:
//   - application/x-www-form-urlencoded  → URL-decoded, field replaced, re-encoded
//   - application/json                   → field replaced in JSON object
//
// Returns the rewritten body. If the body does not contain a "resource" field
// equal to proxyBase, the original body is returned unchanged.
func (r *OAuthRewriter) RewriteResourceParamToUpstream(body []byte, contentType string) ([]byte, error) {
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return r.rewriteFormResourceParam(body)
	}
	if strings.Contains(contentType, "application/json") {
		return r.rewriteJSONResourceParam(body)
	}
	// Unknown content type — return unchanged.
	return body, nil
}

func (r *OAuthRewriter) rewriteFormResourceParam(body []byte) ([]byte, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: parse form body: %w", err)
	}

	if values.Get("resource") == r.proxyBase {
		values.Set("resource", r.upstreamOrigin)
	}

	return []byte(values.Encode()), nil
}

func (r *OAuthRewriter) rewriteJSONResourceParam(body []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("mcp oauth rewriter: parse json body: %w", err)
	}

	if existing, ok := raw["resource"]; ok {
		var resourceVal string
		if err := json.Unmarshal(existing, &resourceVal); err == nil && resourceVal == r.proxyBase {
			upstreamJSON, _ := json.Marshal(r.upstreamOrigin)
			raw["resource"] = upstreamJSON
		}
	}

	return json.Marshal(raw)
}

// RewriteResourceParamInQuery rewrites the "resource" query parameter in a URL
// from proxyBase to upstreamOrigin. Used when token/register requests carry the
// resource value in the query string rather than the body.
//
// Returns the rewritten URL string. If the parameter is absent or does not
// match proxyBase, the original URL is returned unchanged.
func (r *OAuthRewriter) RewriteResourceParamInQuery(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("mcp oauth rewriter: parse url %q: %w", rawURL, err)
	}

	q := u.Query()
	if q.Get("resource") == r.proxyBase {
		q.Set("resource", r.upstreamOrigin)
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}
