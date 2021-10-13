package producer

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/content"
	"github.com/avenga/couper/telemetry"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend http.RoundTripper
	Context hcl.Body
	Name    string // label
}

// Requests represents the producer <Requests> object.
type Requests []*Request

func (r Requests) Produce(ctx context.Context, req *http.Request, results chan<- *Result) {
	var currentName string // at least pre roundtrip
	wg := &sync.WaitGroup{}
	roundtripCreated := false

	var rootSpan trace.Span
	if len(r) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "requests", trace.WithSpanKind(trace.SpanKindProducer))
	}

	defer func() {
		if rp := recover(); rp != nil {
			sendResult(ctx, results, &Result{
				Err: ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				},
				RoundTripName: currentName,
			})

			if !roundtripCreated {
				close(results)
			}
		}
	}()

	evalctx := ctx.Value(request.ContextType).(*eval.Context)
	updated := evalctx.WithClientRequest(req)

	for _, or := range r {
		// span end by result reader
		outCtx, span := telemetry.NewSpanFromContext(withRoundTripName(ctx, or.Name), or.Name, trace.WithSpanKind(trace.SpanKindClient))

		bodyContent, _, diags := or.Context.PartialContent(config.Request{Remain: or.Context}.Schema(true))
		if diags.HasErrors() {
			sendResult(outCtx, results, &Result{Err: diags})
			continue
		}

		method, err := content.GetAttribute(updated.HCLContext(), bodyContent, "method")
		if err != nil {
			sendResult(outCtx, results, &Result{Err: err})
			continue
		}

		body, defaultContentType, err := eval.GetBody(updated.HCLContext(), bodyContent)
		if err != nil {
			sendResult(outCtx, results, &Result{Err: err})
			continue
		}

		url, err := content.GetAttribute(updated.HCLContext(), bodyContent, "url")
		if err != nil {
			sendResult(outCtx, results, &Result{Err: err})
			continue
		}

		if url != "" {
			outCtx = context.WithValue(outCtx, request.URLAttribute, url)
		}

		if method == "" {
			method = http.MethodGet

			if len(body) > 0 {
				method = http.MethodPost
			}
		}

		// The real URL is configured later in the backend,
		// see <go roundtrip()> at the end of current for-loop.
		outreq, err := http.NewRequest(strings.ToUpper(method), "", nil)
		if err != nil {
			sendResult(outCtx, results, &Result{Err: err})
			continue
		}

		if defaultContentType != "" {
			outreq.Header.Set("Content-Type", defaultContentType)
		}

		eval.SetBody(outreq, []byte(body))

		*outreq = *outreq.WithContext(outCtx)
		err = eval.ApplyRequestContext(outCtx, or.Context, outreq)
		if err != nil {
			sendResult(outCtx, results, &Result{Err: err})
			continue
		}

		span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(outreq)...)

		roundtripCreated = true
		wg.Add(1)
		go roundtrip(or.Backend, outreq, results, wg)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
