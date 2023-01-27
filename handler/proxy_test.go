package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/request"
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
		false,
		logEntry,
	)

	outreq := httptest.NewRequest("GET", "https://1.2.3.4/", nil)
	outreq.Header.Set("Authorization", "Basic 123")
	outreq.Header.Set("Cookie", "123")
	outreq = outreq.WithContext(eval.NewContext(nil, &config.Defaults{}, "").WithClientRequest(outreq))
	ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), request.RoundTripProxy, true), time.Now().Add(time.Millisecond*50))
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

func TestProxy_WebsocketsAllowed(t *testing.T) {
	log, _ := test.NewLogger()
	logEntry := log.WithContext(context.Background())

	origin := test.NewBackend()

	pNotAllowed := handler.NewProxy(
		transport.NewBackend(body.NewHCLSyntaxBodyWithStringAttr("origin", origin.Addr()), &transport.Config{
			Origin: origin.Addr(),
		}, nil, logEntry),
		&hclsyntax.Body{},
		false,
		logEntry,
	)

	pAllowed := handler.NewProxy(
		transport.NewBackend(body.NewHCLSyntaxBodyWithStringAttr("origin", origin.Addr()), &transport.Config{
			Origin: origin.Addr(),
		}, nil, logEntry),
		&hclsyntax.Body{},
		true,
		logEntry,
	)

	headers := http.Header{
		"Connection": []string{"upgrade"},
		"Upgrade":    []string{"websocket"},
	}

	outreqN := httptest.NewRequest("GET", "http://couper.local/ws", nil)
	outreqA := httptest.NewRequest("GET", "http://couper.local/ws", nil)

	outCtx := context.WithValue(context.Background(), request.RoundTripProxy, true)

	for _, r := range []*http.Request{outreqN, outreqA} {
		for h := range headers {
			r.Header.Set(h, headers.Get(h))
		}
	}

	resN, _ := pNotAllowed.RoundTrip(outreqN.WithContext(outCtx))
	resA, _ := pAllowed.RoundTrip(outreqA.WithContext(outCtx))

	if resN.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a bad request on ws endpoint without related headers, got: %d", resN.StatusCode)
	}

	if resA.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("expcted passed Connection and Upgrade header which results in 101, got: %d", resA.StatusCode)
	}
}
