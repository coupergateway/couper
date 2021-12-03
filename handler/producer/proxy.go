package producer

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/telemetry"
)

type Proxy struct {
	Name      string // label
	RoundTrip http.RoundTripper
}

type Proxies []*Proxy

func (pr Proxies) Produce(ctx context.Context, clientReq *http.Request, results chan<- *Result) {
	var currentName string // at least pre roundtrip
	wg := &sync.WaitGroup{}
	roundtripCreated := false

	var rootSpan trace.Span
	if len(pr) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "proxies", trace.WithSpanKind(trace.SpanKindProducer))
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

			if !roundtripCreated {
				close(results)
			}
		}
	}()

	for _, proxy := range pr {
		currentName = proxy.Name
		outCtx := withRoundTripName(ctx, proxy.Name)
		outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)

		// span end by result reader
		outCtx, _ = telemetry.NewSpanFromContext(outCtx, proxy.Name, trace.WithSpanKind(trace.SpanKindServer))

		// since proxy and backend may work on the "same" outReq this must be cloned.
		outReq := clientReq.Clone(outCtx)

		roundtripCreated = true
		wg.Add(1)
		go roundtrip(proxy.RoundTrip, outReq, results, wg)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	wg.Wait()
	close(results)
}

func (pr Proxies) Len() int {
	return len(pr)
}
