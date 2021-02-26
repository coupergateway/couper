package producer

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend http.RoundTripper
	Body    string
	Context hcl.Body
	// Dispatch bool
	Method string
	Name   string // label
	URL    string
}

// Requests represents the producer <Requests> object.
type Requests []*Request

func (r Requests) Produce(ctx context.Context, _ *http.Request, evalCtx *hcl.EvalContext, results chan<- *Result) {
	wg := &sync.WaitGroup{}
	wg.Add(len(r))
	go func() {
		wg.Wait()
		close(results)
	}()

	for _, req := range r {
		outreq, err := http.NewRequest(req.Method, req.URL, strings.NewReader(req.Body))
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		*outreq = *withRoundTripName(req.Name, outreq)
		err = eval.ApplyRequestContext(evalCtx, req.Context, outreq)
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}
		*outreq = *outreq.WithContext(ctx)
		go roundtrip(req.Backend, outreq, results, wg)
	}
}

func withRoundTripName(name string, outreq *http.Request) *http.Request {
	n := name
	if n == "" {
		n = "default"
	}
	return outreq.WithContext(context.WithValue(outreq.Context(), request.RoundTripName, n))
}
