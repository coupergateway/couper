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
	Context hcl.Body
	Name    string // label
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
		method, err := eval.GetContextAttribute(or.Context, req, "method")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}
		body, err := eval.GetContextAttribute(or.Context, req, "body")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}
		url, err := eval.GetContextAttribute(or.Context, req, "url")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		if method == "" {
			method = http.MethodGet
		}
		if len(body) > 0 {
			method = http.MethodPost
		}

		// The real URL is configured later in the backend,
		// see <go roundtrip()> on the end of current for-loop.
		outreq, err := http.NewRequest(method, "https://", strings.NewReader(body))
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

		if url != "" {
			outCtx = context.WithValue(outCtx, request.URLAttribute, url)
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
