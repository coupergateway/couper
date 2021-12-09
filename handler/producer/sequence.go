package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/telemetry"
)

type SequenceItem struct {
	Backend http.RoundTripper
	Context hcl.Body
	Name    string // label
}

// Sequence represents a list of serialized items.
type Sequence []*SequenceItem

// Sequences holds several list of serialized items.
type Sequences []Sequence

func (seqs Sequences) Produce(req *http.Request, results chan<- *Result) {
	var rootSpan trace.Span
	ctx := req.Context()
	if len(seqs) > 0 {
		ctx, rootSpan = telemetry.NewSpanFromContext(ctx, "sequences", trace.WithSpanKind(trace.SpanKindProducer))
	}

	resultsCh := make(chan *Result, seqs.Len())

	for _, s := range seqs {
		go s.Produce(req, resultsCh)
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
		outCtx := evalCtx.WithBeresps(lastBeresps...)
		outreq := req.
			WithContext(context.WithValue(outCtx, request.BufferOptions, req.Context().Value(request.BufferOptions)))

		if seq.Context == nil {
			Proxies{&Proxy{Name: seq.Name,
				RoundTrip: seq.Backend,
			}}.Produce(outreq, result)
		} else {
			Requests{&Request{
				Backend: seq.Backend,
				Context: seq.Context,
				Name:    seq.Name,
			}}.Produce(outreq, result)
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
			lastResult.Err = errors.Sequence.With(lastResult.Err)
			results <- lastResult
			return
		}

		lastBeresps = append(lastBeresps, lastResult.Beresp)
	}

	if rootSpan != nil {
		rootSpan.End()
	}

	if lastResult == nil {
		results <- &Result{Err: errors.Sequence.With(errors.New().Message("no result"))}
	}

	results <- lastResult
}

func (s Sequence) Len() int {
	if len(s) > 0 {
		return 1
	}
	return 0
}
