package producer

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/avenga/couper/config/request"
)

type Proxy struct {
	Name      string // label
	RoundTrip http.RoundTripper
}

type Proxies []*Proxy

func (pr Proxies) Produce(ctx context.Context, clientReq *http.Request, results chan<- *Result) {
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

	for _, proxy := range pr {
		currentName = proxy.Name
		outCtx := withRoundTripName(ctx, proxy.Name)
		outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)
		outReq := clientReq.WithContext(outCtx)

		wg.Add(1)
		go roundtrip(proxy.RoundTrip, outReq, results, wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}
