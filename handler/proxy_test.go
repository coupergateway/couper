package handler_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/test"
)

func TestProxy_BlacklistHeaderRemoval(t *testing.T) {
	log, _ := test.NewLogger()
	logEntry := log.WithContext(context.Background())
	p := handler.NewProxy(
		transport.NewBackend(body.NewHCLSyntaxBodyWithStringAttr("origin", "https://1.2.3.4"), &transport.Config{
			Origin: "https://1.2.3.4/",
		}, nil, logEntry),
		&hclsyntax.Body{},
		logEntry,
	)

	outreq := httptest.NewRequest("GET", "https://1.2.3.4/", nil)
	outreq.Header.Set("Authorization", "Basic 123")
	outreq.Header.Set("Cookie", "123")
	outreq = outreq.WithContext(eval.NewContext(nil, &config.Defaults{}, "").WithClientRequest(outreq))
	ctx, cancel := context.WithDeadline(outreq.Context(), time.Now().Add(time.Millisecond*50))
	outreq = outreq.WithContext(ctx)
	defer cancel()

	_, _ = p.RoundTrip(outreq)

	if outreq.Header.Get("Authorization") != "" {
		t.Error("Expected removed Authorization header")
	}

	if outreq.Header.Get("Cookie") != "" {
		t.Error("Expected removed Cookie header")
	}
}
