package producer

import (
	"net/http"
	"sync"
)

// Result represents the producer <Result> object.
type Result struct {
	Beresp *http.Response
	Err    error
	// TODO: trace
}

// Results represents the producer <Result> channel.
type Results chan *Result

func roundtrip(rt http.RoundTripper, req *http.Request, results chan<- *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: apply evals here with context?
	beresp, err := rt.RoundTrip(req)
	results <- &Result{Beresp: beresp, Err: err}
}
