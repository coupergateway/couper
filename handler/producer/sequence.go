package producer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

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

	defer close(results)

	result := make(chan *Result, 1)
	outreq := req
	var lastResult *Result
	var moreEntries bool
	for _, seqReq := range s {
		Requests{seqReq}.Produce(ctx, outreq, result)
		select {
		case <-outreq.Context().Done():
			return // TODO: handle results chan?
		case lastResult, moreEntries = <-result:
			if !moreEntries {
				return
			}
		}

		if lastResult.Err != nil {
			results <- lastResult // TODO: seq error
			return
		}
		// update eval context
		evalCtx := eval.ContextFromRequest(outreq)
		*outreq = *outreq.WithContext(evalCtx.WithBeresps(lastResult.Beresp))
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	// TODO: error handling
	if lastResult == nil {
		return
	}

	results <- lastResult
}

func (s Sequence) Len() int {
	if len(s) > 0 {
		return 1
	}
	return 0
}
