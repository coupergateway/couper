package mcp

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// OAuthHandler is an http.Handler that transparently proxies the four OAuth
// endpoints required for MCP OAuth flows:
//
//	GET  /.well-known/oauth-protected-resource   → fetch from upstream, rewrite response
//	GET  /.well-known/oauth-authorization-server → fetch from upstream, rewrite response
//	POST /token                                  → forward to upstream, rewrite resource param
//	POST /register                               → forward to upstream, rewrite resource param
//
// The proxy base URL is derived from the incoming request's Host header and
// scheme at request time, so the handler works regardless of where Couper runs
// (localhost, k8s, behind a load balancer, custom domain).
//
// All rewriting is delegated to OAuthRewriter which has no dependency on
// http.Request or http.Response.
type OAuthHandler struct {
	upstreamOrigin string // e.g. "https://mcp.example.com"
	mcpEndpoint    string // e.g. "/mcp" — the endpoint pattern for the MCP proxy
	client         *http.Client
	logger         *logrus.Entry
}

// NewOAuthHandler creates an OAuthHandler.
// upstreamOrigin is the origin of the upstream MCP server (e.g. "https://mcp.example.com").
// mcpEndpoint is the endpoint path where the MCP proxy is mounted (e.g. "/mcp").
func NewOAuthHandler(upstreamOrigin, mcpEndpoint string, logger *logrus.Entry) *OAuthHandler {
	return &OAuthHandler{
		upstreamOrigin: strings.TrimRight(upstreamOrigin, "/"),
		mcpEndpoint:    mcpEndpoint,
		client:         &http.Client{},
		logger:         logger,
	}
}

// proxyBaseFromRequest derives the public-facing proxy base URL from the
// incoming request's Host header and TLS state.
func (h *OAuthHandler) proxyBaseFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Also check X-Forwarded-Proto for reverse-proxy setups.
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + r.Host
}

// rewriterFromRequest creates an OAuthRewriter using the proxy base derived
// from the current request. The rewriter is lightweight (two strings).
func (h *OAuthHandler) rewriterFromRequest(r *http.Request) (*OAuthRewriter, error) {
	proxyBase := h.proxyBaseFromRequest(r)
	return NewOAuthRewriter(proxyBase+h.mcpEndpoint, h.upstreamOrigin)
}

// ServeHTTP dispatches to the appropriate sub-handler based on the request path.
// Paths with a trailing sub-path (e.g. /.well-known/oauth-protected-resource/mcp)
// are also handled. Unknown paths return 404.
func (h *OAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	switch {
	case p == "/.well-known/oauth-protected-resource" ||
		strings.HasPrefix(p, "/.well-known/oauth-protected-resource/"):
		h.serveProtectedResourceMetadata(w, r)
	case p == "/.well-known/oauth-authorization-server" ||
		strings.HasPrefix(p, "/.well-known/oauth-authorization-server/"):
		h.serveAuthorizationServerMetadata(w, r)
	case p == "/token":
		h.proxyOAuthEndpoint(w, r, "/token")
	case p == "/register":
		h.proxyOAuthEndpoint(w, r, "/register")
	default:
		http.NotFound(w, r)
	}
}

func (h *OAuthHandler) serveProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	upstreamURL := h.upstreamOrigin + "/.well-known/oauth-protected-resource"

	body, statusCode, contentType, err := h.fetchUpstream(r, upstreamURL)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: fetch protected-resource metadata failed")
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}

	rw, err := h.rewriterFromRequest(r)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: create rewriter failed")
		h.writeJSONResponse(w, statusCode, contentType, body)
		return
	}

	rewritten, err := rw.RewriteProtectedResourceMetadata(body)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: rewrite protected-resource metadata failed")
		rewritten = body
	}

	h.writeJSONResponse(w, statusCode, contentType, rewritten)
}

func (h *OAuthHandler) serveAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	upstreamURL := h.upstreamOrigin + "/.well-known/oauth-authorization-server"

	body, statusCode, contentType, err := h.fetchUpstream(r, upstreamURL)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: fetch authorization-server metadata failed")
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}

	rw, err := h.rewriterFromRequest(r)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: create rewriter failed")
		h.writeJSONResponse(w, statusCode, contentType, body)
		return
	}

	rewritten, err := rw.RewriteAuthorizationServerMetadata(body)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: rewrite authorization-server metadata failed")
		rewritten = body
	}

	h.writeJSONResponse(w, statusCode, contentType, rewritten)
}

func (h *OAuthHandler) proxyOAuthEndpoint(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	rw, err := h.rewriterFromRequest(r)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: create rewriter failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	upstreamURLStr, err := rw.RewriteResourceParamInQuery(
		h.upstreamOrigin + upstreamPath + "?" + r.URL.RawQuery,
	)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: rewrite query resource param failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var reqBody []byte
	if r.Body != nil {
		reqBody, err = io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			h.logger.WithError(err).Error("mcp oauth: read request body failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	contentType := r.Header.Get("Content-Type")
	rewrittenBody, err := rw.RewriteResourceParamToUpstream(reqBody, contentType)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: rewrite body resource param failed")
		rewrittenBody = reqBody
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURLStr, bytes.NewReader(rewrittenBody))
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: create upstream request failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	copyHeaders(upstreamReq.Header, r.Header)
	upstreamReq.Header.Set("Content-Type", contentType)
	upstreamReq.ContentLength = int64(len(rewrittenBody))

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: upstream request failed")
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.WithError(err).Error("mcp oauth: read upstream response failed")
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(respBody)))
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func (h *OAuthHandler) fetchUpstream(r *http.Request, upstreamURL string) ([]byte, int, string, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		return nil, 0, "", err
	}

	if accept := r.Header.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", err
	}

	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func (h *OAuthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, contentType string, body []byte) {
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func copyHeaders(dst, src http.Header) {
	skipHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
		"Content-Length":      true,
		"Content-Type":        true,
	}

	for k, vv := range src {
		if skipHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// OAuthPaths lists the URL paths that OAuthHandler serves. Used by the runtime
// to automatically register additional routes when beta_mcp_proxy is configured.
var OAuthPaths = []string{
	"/.well-known/oauth-protected-resource",
	"/.well-known/oauth-protected-resource/**",
	"/.well-known/oauth-authorization-server",
	"/.well-known/oauth-authorization-server/**",
	"/token",
	"/register",
}

// IsOAuthPath reports whether path is one of the paths served by OAuthHandler.
func IsOAuthPath(p string) bool {
	clean := strings.TrimRight(p, "/")
	for _, known := range OAuthPaths {
		trimmed := strings.TrimRight(known, "/*")
		if clean == trimmed || strings.HasPrefix(clean, trimmed+"/") {
			return true
		}
	}
	return false
}
