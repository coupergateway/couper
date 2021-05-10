package producer

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

var (
	_ Roundtrips = Proxies{}
	_ Roundtrips = Requests{}
)

type Roundtrips interface {
	Produce(ctx context.Context, req *http.Request, results chan<- *Result, log *logrus.Entry)
}

func sendResult(ctx context.Context, results chan<- *Result, result *Result) {
	select {
	case <-ctx.Done():
		return
	case results <- result:
	}
}
