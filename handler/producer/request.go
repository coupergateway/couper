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
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/telemetry"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend          http.RoundTripper
	Context          hcl.Body
	Name             string // label
	PreviousSequence string
}

// Requests represents the producer <Requests> object.
type Requests []*Request

func (r Requests) Produce(req *http.Request, results chan<- *Result) {
	var currentName string // at least pre roundtrip
	wg := &sync.WaitGroup{}
	ctx := req.Context()
	var rootSpan trace.Span
	if r.Len() > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "requests", trace.WithSpanKind(trace.SpanKindProducer))
	}

	defer func() {
		if rp := recover(); rp != nil {
			results <- &Result{
				Err: ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				},
				RoundTripName: currentName,
			}
		}
	}()

	hclCtx := eval.ContextFromRequest(req).HCLContextSync() // also synced for requests due to sequence case

	for _, or := range r {
		// span end by result reader
		outCtx, span := telemetry.NewSpanFromContext(withRoundTripName(ctx, or.Name), or.Name, trace.WithSpanKind(trace.SpanKindClient))
		if or.PreviousSequence != "" {
			outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, or.PreviousSequence)
		}

		bodyContent, _, diags := or.Context.PartialContent(config.Request{Remain: or.Context}.Schema(true))
		if diags.HasErrors() {
			results <- &Result{Err: diags}
			continue
		}

		methodVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "method")
		if err != nil {
			results <- &Result{Err: err}
			continue
		}
		method := seetie.ValueToString(methodVal)

		url, err := NewURLFromAttribute(hclCtx, or.Context, "url", req)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		body, defaultContentType, err := eval.GetBody(hclCtx, bodyContent)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		if method == "" {
			method = http.MethodGet

			if len(body) > 0 {
				method = http.MethodPost
			}
		}

		outreq, err := http.NewRequest(strings.ToUpper(method), url.String(), nil)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		expStatusVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "expected_status")
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		outCtx = context.WithValue(outCtx, request.EndpointExpectedStatus, seetie.ValueToIntSlice(expStatusVal))

		if defaultContentType != "" {
			outreq.Header.Set("Content-Type", defaultContentType)
		}

		eval.SetBody(outreq, []byte(body))

		outreq = outreq.WithContext(outCtx)
		err = eval.ApplyRequestContext(hclCtx, or.Context, outreq)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(outreq)...)

		wg.Add(1)
		go roundtrip(or.Backend, outreq, results, wg)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	wg.Wait()
}

func (r Requests) Len() int {
	return len(r)
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
