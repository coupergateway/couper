package producer

import (
	"context"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/telemetry"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend          http.RoundTripper
	Context          *hclsyntax.Body
	Name             string // label
	previousSequence string
}

func (r *Request) Produce(req *http.Request) *Result {
	ctx := req.Context()
	var rootSpan trace.Span
	ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "requests", trace.WithSpanKind(trace.SpanKindProducer))

	hclCtx := eval.ContextFromRequest(req).HCLContextSync() // also synced for requests due to sequence case

	// span end by result reader
	outCtx, span := telemetry.NewSpanFromContext(withRoundTripName(ctx, r.Name), r.Name, trace.WithSpanKind(trace.SpanKindClient))
	if r.previousSequence != "" {
		outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, r.previousSequence)
	}

	methodVal, err := eval.ValueFromBodyAttribute(hclCtx, r.Context, "method")
	if err != nil {
		return &Result{Err: err}
	}
	method := seetie.ValueToString(methodVal)

	outreq := req.Clone(req.Context())
	removeHost(outreq)

	url, err := NewURLFromAttribute(hclCtx, r.Context, "url", outreq)
	if err != nil {
		return &Result{Err: err}
	}

	body, defaultContentType, err := eval.GetBody(hclCtx, r.Context)
	if err != nil {
		return &Result{Err: err}
	}

	if method == "" {
		method = http.MethodGet

		if len(body) > 0 {
			method = http.MethodPost
		}
	}

	outreq, err = http.NewRequest(strings.ToUpper(method), url.String(), nil)
	if err != nil {
		return &Result{Err: err}
	}

	expStatusVal, err := eval.ValueFromBodyAttribute(hclCtx, r.Context, "expected_status")
	if err != nil {
		return &Result{Err: err}
	}

	outCtx = context.WithValue(outCtx, request.EndpointExpectedStatus, seetie.ValueToIntSlice(expStatusVal))

	if defaultContentType != "" {
		outreq.Header.Set("Content-Type", defaultContentType)
	}

	eval.SetBody(outreq, []byte(body))

	outreq = outreq.WithContext(outCtx)
	err = eval.ApplyRequestContext(hclCtx, r.Context, outreq)
	if err != nil {
		return &Result{Err: err}
	}

	span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(outreq)...)

	result := roundtrip(r.Backend, outreq)

	rootSpan.End()
	return result
}

func (r *Request) SetPreviousSequence(ps string) {
	r.previousSequence = ps
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
