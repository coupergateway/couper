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
	var currentName string
	var roundtrips int
	wg := &sync.WaitGroup{}

	defer func() {
		if rp := recover(); rp != nil {
			results <- &Result{
				Err: ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				},
				RoundTripName: currentName,
			}
		}

		if roundtrips == 0 {
			close(results)
		} else {
			go func() {
				wg.Wait()
				close(results)
			}()
		}
	}()

	for _, proxy := range pr {
		currentName = proxy.Name
		outCtx := withRoundTripName(ctx, proxy.Name)
		outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)
		outReq := clientReq.WithContext(outCtx)
		roundtrips++
		go roundtrip(proxy.RoundTrip, outReq, results, wg)
	}
}
