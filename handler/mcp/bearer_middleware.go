package mcp

import (
	"context"
	"net/http"

	"github.com/coupergateway/couper/config/request"
)

// BearerTokenRoundTripper wraps another RoundTripper, saving the incoming
// Authorization header into the request context before delegating. This
// preserves the token before Proxy's headerBlacklist strips it.
// MCPRoundTripper then restores the header from context.
type BearerTokenRoundTripper struct {
	next http.RoundTripper
}

// NewBearerTokenRoundTripper wraps next so the Authorization header is
// captured into the request context before it is stripped downstream.
func NewBearerTokenRoundTripper(next http.RoundTripper) *BearerTokenRoundTripper {
	return &BearerTokenRoundTripper{next: next}
}

func (b *BearerTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if auth := req.Header.Get("Authorization"); auth != "" {
		ctx := context.WithValue(req.Context(), request.MCPBearerToken, auth)
		req = req.WithContext(ctx)
	}
	return b.next.RoundTrip(req)
}
