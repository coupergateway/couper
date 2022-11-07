package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/telemetry"
)

type SequenceItem struct {
	Backend http.RoundTripper
	Context hcl.Body
	Name    string // label
}

// Sequence represents a list of serialized items.
type Sequence []Roundtrip

// Sequences holds several list of serialized items which effectively gets executed in parallel.
type Sequences []Roundtrip

func (seqs Sequences) Produce(req *http.Request) chan *Result {
	return pipe(req, seqs, "sequences")
}

func (seqs Sequences) Len() int {
	return len(seqs)
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

// pipe calls the Roundtrip Interface on each given item and distinguish between parallelism and trace kind.
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
		case "sequences": // execute each sequence branch in parallel
			go pipeResult(ctx, req, rch, srt)
		case "sequence": // one by one
			pipeResult(ctx, req, rch, srt)
		}
	}

	// Since the sequence gets resolved in order just the last item matters.
	for _, rch := range allResults {
		var last *Result
		for last = range rch {
			// drain
		}
		result <- last
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
