package handler

import (
	"net/http/httptest"
	"testing"
)

// TestProxy_director checks for a working blacklist header removal
// within handler.Proxy director func.
func TestProxy_director(t *testing.T) {
	p := &Proxy{}

	outreq := httptest.NewRequest("GET", "https://couper.io/", nil)
	outreq.Header.Set("Authorization", "Basic 123")
	outreq.Header.Set("Cookie", "123")

	p.director(outreq)

	if outreq.Header.Get("Authorization") != "" {
		t.Error("Expected removed Authorization header")
	}

	if outreq.Header.Get("Cookie") != "" {
		t.Error("Expected removed Cookie header")
	}
}
