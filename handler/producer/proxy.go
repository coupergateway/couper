package producer

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/telemetry"
)

type Proxy struct {
	Content          hcl.Body
	Name             string // label
	PreviousSequence string
	RoundTrip        http.RoundTripper
}

type Proxies []*Proxy

func (pr Proxies) Produce(clientReq *http.Request) chan *Result {
	var currentName string // at least pre roundtrip
	wg := &sync.WaitGroup{}
	ctx := clientReq.Context()
	var rootSpan trace.Span
	if len(pr) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "proxies", trace.WithSpanKind(trace.SpanKindProducer))
	}

	results := make(chan *Result, pr.Len())
	defer close(results)

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

	for _, proxy := range pr {
		currentName = proxy.Name
		outCtx := withRoundTripName(ctx, proxy.Name)
		outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)
		if proxy.PreviousSequence != "" {
			outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, proxy.PreviousSequence)
		}

		// span end by result reader
		outCtx, _ = telemetry.NewSpanFromContext(outCtx, proxy.Name, trace.WithSpanKind(trace.SpanKindServer))

		// since proxy and backend may work on the "same" outReq this must be cloned.
		outReq := clientReq.Clone(outCtx)
		removeHost(outReq)

		hclCtx := eval.ContextFromRequest(clientReq).HCLContext()
		url, err := NewURLFromAttribute(hclCtx, proxy.Content, "url", outReq)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}

		// proxy should pass query if not redefined with url attribute
		if outReq.URL.RawQuery != "" && url.RawQuery == "" {
			url.RawQuery = outReq.URL.RawQuery
		}

		outReq.URL = url

		wg.Add(1)
		go roundtrip(proxy.RoundTrip, outReq, results, wg)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	wg.Wait()

	return results
}

func (pr Proxies) Len() int {
	return len(pr)
}
