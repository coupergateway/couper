package producer

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/hashicorp/hcl/v2"
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
	var currentName string // at least pre roundtrip
	wg := &sync.WaitGroup{}

	defer func() {
		if rp := recover(); rp != nil {
			sendResult(ctx, results, &Result{
				Err: ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				},
				RoundTripName: currentName,
			})
		}
	}()

	evalctx := ctx.Value(eval.ContextType).(*eval.Context)
	updated := evalctx.WithClientRequest(req)

	for _, or := range r {
		outCtx := withRoundTripName(ctx, or.Name)

		content, _, diags := or.Context.PartialContent(config.Request{Remain: or.Context}.Schema(true))
		if diags.HasErrors() {
			sendResult(ctx, results, &Result{Err: diags})
			continue
		}

		method, err := eval.GetAttribute(updated.HCLContext(), content, "method")
		if err != nil {
			sendResult(ctx, results, &Result{Err: err})
			continue
		}

		body, defaultContentType, err := eval.GetBody(updated.HCLContext(), content)
		if err != nil {
			sendResult(ctx, results, &Result{Err: err})
			continue
		}

		url, err := eval.GetAttribute(updated.HCLContext(), content, "url")
		if err != nil {
			sendResult(ctx, results, &Result{Err: err})
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
			sendResult(ctx, results, &Result{Err: err})
			continue
		}

		if defaultContentType != "" {
			outreq.Header.Set("Content-Type", defaultContentType)
		}

		*outreq = *outreq.WithContext(outCtx)
		err = eval.ApplyRequestContext(outCtx, or.Context, outreq)
		if err != nil {
			sendResult(ctx, results, &Result{Err: err})
			continue
		}

		wg.Add(1)
		go roundtrip(or.Backend, outreq, results, wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
