package producer

import (
	"context"
	"net/http"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
)

// Proxies represents a list of producer <Proxy> objects.
type Proxies []http.RoundTripper

func (pr Proxies) Produce(ctx context.Context, clientReq *http.Request, evalCtx *hcl.EvalContext, results chan<- *Result) {
	wg := &sync.WaitGroup{}
	wg.Add(len(pr))
	go func() {
		wg.Wait()
		close(results)
	}()

	for _, proxy := range pr {
		outreq := clientReq.WithContext(ctx)
		// TODO: proxy ctx
		//err := eval.ApplyRequestContext(evalCtx, proxy.Context, outreq)
		err := eval.ApplyRequestContext(evalCtx, nil, outreq)
		if err != nil {
			results <- &Result{Err: err}
			wg.Done()
			continue
		}
		//go roundtrip(proxy, outreq, evalCtx, proxy.Context, results, wg)
		go roundtrip(proxy, outreq, evalCtx, nil, results, wg)
	}
}
