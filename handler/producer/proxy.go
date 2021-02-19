package producer

import (
	"context"
	"net/http"
	"sync"

	"github.com/hashicorp/hcl/v2"
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
		go roundtrip(proxy, outreq, results, wg)
	}
}
