package producer

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/telemetry"
)

// Sequence holds a list of items which get executed sequentially.
type Sequence []Roundtrip

// Parallel holds a list of items which get executed in parallel.
type Parallel []Roundtrip

func (p Parallel) Produce(req *http.Request, additionalSync *sync.Map) chan *Result {
	return pipe(req, p, "parallel", additionalSync)
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

func (s Sequence) Produce(req *http.Request, additionalSync *sync.Map) chan *Result {
	return pipe(req, s, "sequence", additionalSync)
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
func pipe(req *http.Request, rt []Roundtrip, kind string, additionalSync *sync.Map) chan *Result {
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
		k := fmt.Sprintf("%v", srt.Names())
		if val, ok := additionalSync.Load(k); ok {
			additionalChs := val.([]chan *Result)
			// srt is already prepared to Produce(), so we can here just listen to the result(s)
			ach := make(chan *Result, srt.Len())
			additionalChs = append(additionalChs, ach)
			additionalSync.Store(k, additionalChs)
			switch kind {
			case "parallel": // execute each sequence branch in parallel
				go func() {
					pipeResults(rch, ach)
					close(rch)
				}()
			case "sequence": // one by one
				pipeResults(rch, ach)
				close(rch)
			}
			continue
		}
		additionalSync.Store(k, []chan *Result{})

		switch kind {
		case "parallel": // execute each sequence branch in parallel
			go produceAndPipeResults(ctx, req, rch, srt, additionalSync)
		case "sequence": // one by one
			produceAndPipeResults(ctx, req, rch, srt, additionalSync)
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

func pipeResults(target, src chan *Result) {
	for r := range src {
		target <- r
	}
}

func produceAndPipeResults(ctx context.Context, req *http.Request, results chan *Result, rt Roundtrip, additionalSync *sync.Map) {
	outreq := req.WithContext(ctx)
	defer close(results)
	rs := rt.Produce(outreq, additionalSync)

	k := fmt.Sprintf("%v", rt.Names())
	var additionalChs []chan *Result
	if val, ok := additionalSync.Load(k); ok {
		additionalChs = val.([]chan *Result)
	}
	for _, ach := range additionalChs {
		defer close(ach)
	}

	for r := range rs {
		select {
		case <-ctx.Done():
			e := &Result{Err: ctx.Err()}
			results <- e
			for _, ach := range additionalChs {
				ach <- e
			}
			return
		case results <- r:
			for _, ach := range additionalChs {
				ach <- r
			}
		}
	}
}
