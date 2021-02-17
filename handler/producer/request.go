package producer

import (
	"context"
	"io"
	"net/http"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/handler/transport"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend *transport.Backend
	Body    io.Reader
	Context hcl.Body
	// Dispatch bool
	Method string
	Name   string // label
	URL    string
}

// Requests represents the producer <Requests> object.
type Requests struct {
	eval *hcl.EvalContext
	list []*Request
}

func NewRequests(eval *hcl.EvalContext, reqs []*Request, idFormat string) *Requests {
	return &Requests{
		eval: eval,
		list: reqs[:],
	}
}

func (r *Requests) Produce(ctx context.Context, results Results) {
	for _, req := range r.list {
		outreq, err := http.NewRequest(req.Method, req.URL, req.Body)
		if err != nil {
			results <- &Result{Err: err}
			continue
		}
		*outreq = *outreq.WithContext(ctx)

		backend := req.Backend
		go func() {
			beresp, e := backend.RoundTrip(outreq)
			results <- &Result{Beresp: beresp, Err: e}
		}()
	}
}
