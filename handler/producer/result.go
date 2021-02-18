package producer

import (
	"net/http"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
)

// Result represents the producer <Result> object.
type Result struct {
	Beresp *http.Response
	Err    error
	// TODO: trace
}

// Results represents the producer <Result> channel.
type Results chan *Result

func roundtrip(rt http.RoundTripper, req *http.Request, evalCtx *hcl.EvalContext, hclContext hcl.Body, results chan<- *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	beresp, err := rt.RoundTrip(req)
	if err != nil {
		results <- &Result{Err: err}
		return
	}

	err = eval.ApplyResponseContext(evalCtx, hclContext, req, beresp)
	results <- &Result{Beresp: beresp, Err: err}
}
