package producer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/telemetry"
)

// Sequence holds a list of items which get executed sequentially.
type Sequence []Roundtrip

// Parallel holds a list of items which get executed in parallel.
type Parallel []Roundtrip

func (p Parallel) Produce(req *http.Request) chan *Result {
	return pipe(req, p, "parallel")
}

func (p Parallel) Len() int {
	return len(p)
}

func (p Parallel) Names() []string {
	var names []string
	for _, i := range p {
		names = append(names, i.Names()...)
	}
	return names
}

func (s Sequence) Produce(req *http.Request) chan *Result {
	return pipe(req, s, "sequence")
}

func (s Sequence) Len() int {
	var sum int
	for _, t := range s {
		sum += t.Len()
	}
	return sum
}

func (s Sequence) Names() []string {
	var names []string
	for _, i := range s {
		names = append(names, i.Names()...)
	}
	return names
}

// pipe calls the Roundtrip Interface on each given item and distinguishes between parallelism and trace kind.
// The returned channel will be closed if this chain part has been ended.
func pipe(req *http.Request, rt []Roundtrip, kind string) chan *Result {
	var rootSpan trace.Span
	ctx := req.Context()
	if len(rt) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, kind, trace.WithSpanKind(trace.SpanKindProducer))
		defer rootSpan.End()
	}

	result := make(chan *Result, len(rt))
	var allResults []chan *Result

	for _, srt := range rt {
		rch := make(chan *Result, srt.Len())
		allResults = append(allResults, rch)

		switch kind {
		case "parallel": // execute each sequence branch in parallel
			go pipeResult(ctx, req, rch, srt)
		case "sequence": // one by one
			pipeResult(ctx, req, rch, srt)
		}
	}

	// Since the sequence gets resolved in order just the last item matters.
	for _, rch := range allResults {
		var last *Result
		var err error

		for last = range rch {
			// drain
			if last.Err != nil {
				err = last.Err
				// drain must be continued (pipeResult)
			}
		}

		if err != nil {
			result <- &Result{Err: err}
		} else if last == nil {
			result <- &Result{Err: errors.Sequence.Message("no result")}
		} else {
			result <- last
		}
	}

	close(result)
	return result
}

func pipeResult(ctx context.Context, req *http.Request, results chan *Result, rt Roundtrip) {
	outreq := req.WithContext(ctx)
	defer close(results)

	for r := range rt.Produce(outreq) {
		select {
		case <-ctx.Done():
			results <- &Result{Err: ctx.Err()}
			return
		case results <- r:
		}
	}
}
