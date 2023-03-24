package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/telemetry"
)

type Proxy struct {
	Content          *hclsyntax.Body
	Name             string // label
	previousSequence string
	RoundTrip        http.RoundTripper
}

func (p *Proxy) Produce(clientReq *http.Request) *Result {
	ctx := clientReq.Context()
	var rootSpan trace.Span
	ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "proxies", trace.WithSpanKind(trace.SpanKindProducer))

	outCtx := withRoundTripName(ctx, p.Name)
	outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)
	if p.previousSequence != "" {
		outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, p.previousSequence)
	}

	// span end by result reader
	outCtx, _ = telemetry.NewSpanFromContext(outCtx, p.Name, trace.WithSpanKind(trace.SpanKindServer))

	// since proxy and backend may work on the "same" outReq this must be cloned.
	outReq := clientReq.Clone(outCtx)
	removeHost(outReq)

	hclCtx := eval.ContextFromRequest(clientReq).HCLContext()
	url, err := NewURLFromAttribute(hclCtx, p.Content, "url", outReq)
	if err != nil {
		return &Result{Err: err}
	}

	// proxy should pass query if not redefined with url attribute
	if outReq.URL.RawQuery != "" && url.RawQuery == "" {
		url.RawQuery = outReq.URL.RawQuery
	}

	outReq.URL = url

	result := roundtrip(p.RoundTrip, outReq)

	rootSpan.End()
	return result
}

func (p *Proxy) SetPreviousSequence(ps string) {
	p.previousSequence = ps
}
