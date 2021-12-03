package producer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/telemetry"
)

// Sequence represents a list of serialized requests
type Sequence []*Request

func (s Sequence) Produce(ctx context.Context, req *http.Request, results chan<- *Result) {
	var rootSpan trace.Span
	if len(s) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "sequence", trace.WithSpanKind(trace.SpanKindProducer))
	}

	result := make(chan *Result, 1)

	var lastResult *Result
	var lastBeresps []*http.Response
	var moreEntries bool
	for _, seqReq := range s {
		// update eval context
		evalCtx := eval.ContextFromRequest(req)
		outreq := req.WithContext(evalCtx.WithBeresps(lastBeresps...))

		Requests{seqReq}.Produce(ctx, outreq, result)
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
