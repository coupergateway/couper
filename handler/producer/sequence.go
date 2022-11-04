package producer

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/telemetry"
)

type SequenceItem struct {
	Backend http.RoundTripper
	Context hcl.Body
	Name    string // label
}

// Sequence represents a list of serialized items.
type Sequence []Roundtrip

// Sequences holds several list of serialized items.
type Sequences []Roundtrip

func (seqs Sequences) Produce(req *http.Request, results chan<- *Result) {
	var rootSpan trace.Span
	ctx := req.Context()
	if len(seqs) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "sequences", trace.WithSpanKind(trace.SpanKindProducer))
	}

	resultsCh := make(chan *Result)

	for _, s := range seqs {
		outreq := req.WithContext(ctx)
		go s.Produce(outreq, resultsCh)
	}

	for i := 0; i < seqs.Len(); i++ {
		results <- <-resultsCh
	}

	if rootSpan != nil {
		rootSpan.End()
	}
}

func (seqs Sequences) Len() int {
	return len(seqs)
}

func (s Sequence) Produce(req *http.Request, results chan<- *Result) {
	var rootSpan trace.Span

	ctx := req.Context()
	l := s.Len()
	if l > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "sequence", trace.WithSpanKind(trace.SpanKindProducer))
	}
	defer func() {
		if rootSpan != nil {
			rootSpan.End()
		}
	}()

	result := make(chan *Result, l)

	var lastResult *Result
	var lastBeresps []*http.Response

	for _, seq := range s {
		outCtx := ctx

		outreq := req.WithContext(outCtx)

		seq.Produce(outreq, result)

		// handle nested sequences by len; expected results
		for i := 0; i < seq.Len(); i++ {
			select {
			case <-outreq.Context().Done():
				return
			case lastResult = <-result:
			}

			if lastResult.Err != nil {
				if _, ok := lastResult.Err.(*errors.Error); !ok {
					lastResult.Err = errors.Sequence.With(lastResult.Err)
				}
				results <- lastResult
				return
			}

			lastBeresps = append(lastBeresps, lastResult.Beresp)
		}
	}

	if lastResult == nil {
		results <- &Result{Err: errors.Sequence.With(errors.New().Message("no result"))}
	}

	results <- lastResult
}

func (s Sequence) Len() int {
	var sum int
	for _, t := range s {
		sum += t.Len()
	}
	return sum
}
