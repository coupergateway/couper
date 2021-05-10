package producer

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

// Result represents the producer <Result> object.
type Result struct {
	Beresp        *http.Response
	Err           error
	RoundTripName string
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
	defer wg.Done()

	rtn := req.Context().Value(request.RoundTripName).(string)

	defer func() {
		if rp := recover(); rp != nil {
			var err error
			if rp == http.ErrAbortHandler {
				err = errors.Proxy.Message("body copy failed")
			} else {
				err = &ResultPanic{
					err:   fmt.Errorf("%v", rp),
					stack: debug.Stack(),
				}
			}
			sendResult(req.Context(), results, &Result{
				Err:           err,
				RoundTripName: rtn,
			})
		}
	}()

	beresp, err := rt.RoundTrip(req)
	sendResult(req.Context(), results, &Result{
		Beresp:        beresp,
		Err:           err,
		RoundTripName: rtn,
	})
}
