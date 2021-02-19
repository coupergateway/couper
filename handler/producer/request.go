package producer

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend *transport.Backend
	Body    io.Reader
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
		outreq, err := http.NewRequest(req.Method, req.URL, req.Body)
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

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
