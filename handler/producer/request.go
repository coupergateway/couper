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
		outCtx := withRoundTripName(ctx, or.Name)

		method, err := eval.GetContextAttribute(or.Context, outCtx, "method")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		body, err := eval.GetContextAttribute(or.Context, outCtx, "body")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		url, err := eval.GetContextAttribute(or.Context, outCtx, "url")
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		if url != "" {
			outCtx = context.WithValue(outCtx, request.URLAttribute, url)
		}

		if method == "" {
			method = http.MethodGet

			if len(body) > 0 {
				method = http.MethodPost
			}
		}

		// The real URL is configured later in the backend,
		// see <go roundtrip()> at the end of current for-loop.
		outreq, err := http.NewRequest(strings.ToUpper(method), "", strings.NewReader(body))
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

		*outreq = *outreq.WithContext(outCtx)
		err = eval.ApplyRequestContext(outCtx, or.Context, outreq)
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}

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
