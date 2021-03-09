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

type ResultMap map[string]*Result

func (rm ResultMap) List() []*http.Response {
	var list []*http.Response
	for _, br := range rm {
		list = append(list, br.Beresp)
	}
	return list
}

func roundtrip(rt http.RoundTripper, req *http.Request, results chan<- *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: apply evals here with context?
	beresp, err := rt.RoundTrip(req)
	results <- &Result{Beresp: beresp, Err: err}
}
