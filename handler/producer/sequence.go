package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/telemetry"
)

type SequenceItem struct {
	Backend http.RoundTripper
	Context hcl.Body
	Name    string // label
}

// Sequence represents a list of serialized requests
type Sequence []*SequenceItem

func (s Sequence) Produce(ctx context.Context, req *http.Request, results chan<- *Result) {
	var rootSpan trace.Span
	if len(s) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "sequence", trace.WithSpanKind(trace.SpanKindProducer))
	}

	result := make(chan *Result, 1)

	var lastResult *Result
	var lastBeresps []*http.Response
	var moreEntries bool
	for _, seq := range s {
		// update eval context
		evalCtx := eval.ContextFromRequest(req)
		outreq := req.WithContext(evalCtx.WithBeresps(lastBeresps...))

		if seq.Context == nil {
			Proxies{&Proxy{Name: seq.Name,
				RoundTrip: seq.Backend,
			}}.Produce(ctx, outreq, result)
		} else {
			Requests{&Request{
				Backend: seq.Backend,
				Context: seq.Context,
				Name:    seq.Name,
			}}.Produce(ctx, outreq, result)
		}

		select {
		case <-outreq.Context().Done():
			return
		case lastResult, moreEntries = <-result:
			if !moreEntries {
				return
			}
		}

		if lastResult.Err != nil {
			lastResult.Err = errors.Endpoint.Kind("sequence").With(lastResult.Err)
			results <- lastResult
			return
		}

		lastBeresps = append(lastBeresps, lastResult.Beresp)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	if lastResult == nil {
		results <- &Result{Err: errors.Endpoint.Kind("sequence").
			With(errors.New().Message("no result"))}
	}

	results <- lastResult
}

func (s Sequence) Len() int {
	if len(s) > 0 {
		return 1
	}
	return 0
}
