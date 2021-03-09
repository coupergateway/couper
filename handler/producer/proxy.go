package producer

import (
	"context"
	"net/http"
	"sync"
)

type Proxy struct {
	Name      string // label
	RoundTrip http.RoundTripper
}

type Proxies []*Proxy

func (pr Proxies) Produce(ctx context.Context, clientReq *http.Request, results chan<- *Result) {
	wg := &sync.WaitGroup{}
	wg.Add(len(pr))
	go func() {
		wg.Wait()
		close(results)
	}()

	for _, proxy := range pr {
		outCtx := withRoundTripName(ctx, proxy.Name)
		outReq := clientReq.WithContext(outCtx)
		go roundtrip(proxy.RoundTrip, outReq, results, wg)
	}
}
