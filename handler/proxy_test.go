package handler

import (
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval"
)

func TestProxy_BlacklistHeaderRemoval(t *testing.T) {
	p := &Proxy{
		context:      hcl.EmptyBody(),
		reverseProxy: httputil.NewSingleHostReverseProxy(&url.URL{Host: "couper.io", Scheme: "https"}),
	}

	outreq := httptest.NewRequest("GET", "https://1.2.3.4/", nil)
	outreq.Header.Set("Authorization", "Basic 123")
	outreq.Header.Set("Cookie", "123")
	outreq.WithContext(eval.NewContext(nil, &config.Defaults{}).WithClientRequest(outreq))

	_, err := p.RoundTrip(outreq)
	if err != nil {
		t.Error(err)
	}

	if outreq.Header.Get("Authorization") != "" {
		t.Error("Expected removed Authorization header")
	}

	if outreq.Header.Get("Cookie") != "" {
		t.Error("Expected removed Cookie header")
	}
}
