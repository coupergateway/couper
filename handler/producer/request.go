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
	Method  string
	Name    string // label
	URL     string
}

// Requests represents the producer <Requests> object.
type Requests []*Request

func (r Requests) Produce(ctx context.Context, req *http.Request, results chan<- *Result) {
	wg := &sync.WaitGroup{}
	wg.Add(len(r))
	go func() {
		wg.Wait()
		close(results)
	}()

	for _, or := range r {
		outreq, err := http.NewRequest(or.Method, or.URL, strings.NewReader(or.Body))
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		outCtx := withRoundTripName(ctx, or.Name)
		err = eval.ApplyRequestContext(req.Context(), or.Context, outreq)
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}
		*outreq = *outreq.WithContext(outCtx)
		go roundtrip(or.Backend, outreq, results, wg)
	}
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
