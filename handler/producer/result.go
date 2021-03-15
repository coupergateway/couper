package producer

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/avenga/couper/errors"
)

// Result represents the producer <Result> object.
type Result struct {
	Beresp *http.Response
	Err    error
	// TODO: trace
}

type ResultPanic struct {
	err   error
	stack []byte
}

func (r ResultPanic) Error() string {
	return fmt.Sprintf("panic: %v\n%s", r.err, string(r.stack))
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
	defer func() {
		if rp := recover(); rp != nil {
			var err error
			if rp == http.ErrAbortHandler {
				err = errors.EndpointProxyBodyCopyFailed
			} else {
				err = &ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				}
			}
			results <- &Result{Err: err}
		}
		wg.Done()
	}()

	// TODO: apply evals here with context?
	beresp, err := rt.RoundTrip(req)
	results <- &Result{Beresp: beresp, Err: err}
}
