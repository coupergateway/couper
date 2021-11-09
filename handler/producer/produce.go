package producer

import (
	"context"
	"net/http"
)

var (
	_ Roundtrips = Proxies{}
	_ Roundtrips = Requests{}
)

type Roundtrips interface {
	Produce(ctx context.Context, req *http.Request, results chan<- *Result)
}

func sendResult(ctx context.Context, results chan<- *Result, result *Result) {
	select {
	case <-ctx.Done():
		return
	case results <- result:
	}
}
