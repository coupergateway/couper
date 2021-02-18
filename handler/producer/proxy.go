package producer

import (
	"context"
	"net/http"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
)

// Proxy represents the producer <Proxy> object.
type Proxy struct {
	Backend *transport.Backend
	Context hcl.Body
	Proxy   http.Handler
}

// Proxies represents a list of producer <Proxy> objects.
type Proxies []*Proxy

func (p Proxies) Produce(ctx context.Context, clientReq *http.Request, evalCtx *hcl.EvalContext, results chan<- *Result) {
	wg := &sync.WaitGroup{}
	wg.Add(len(p))
	go func() {
		wg.Wait()
		close(results)
	}()

	for _, proxy := range p {
		outreq := clientReq.WithContext(ctx)
		eval.ApplyRequestContext(evalCtx, proxy.Context, outreq)

		backend := proxy.Backend
		hclContext := proxy.Context
		go func() {
			beresp, e := backend.RoundTrip(outreq)
			eval.ApplyResponseContext(evalCtx, hclContext, beresp)
			results <- &Result{Beresp: beresp, Err: e}
			wg.Done()
		}()
	}
}
