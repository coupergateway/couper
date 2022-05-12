package producer

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"go.opentelemetry.io/otel/trace"

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
	span := trace.SpanFromContext(req.Context())

	defer func() {
		if rp := recover(); rp != nil {
			err := errors.Server.With(&ResultPanic{
				err:   fmt.Errorf("%v", rp),
				stack: debug.Stack(),
			})
			span.End()

			results <- &Result{
				Err:           err,
				RoundTripName: rtn,
			}
		}
	}()

	beresp, err := rt.RoundTrip(req)
	span.End()

	if _, ok := err.(*errors.Error); ok {
		results <- &Result{
			Beresp:        beresp,
			Err:           err,
			RoundTripName: rtn,
		}

		return
	}

	if expStatus, ok := req.Context().Value(request.EndpointExpectedStatus).([]int64); beresp != nil &&
		ok && len(expStatus) > 0 {
		var seen bool
		for _, exp := range expStatus {
			if beresp.StatusCode == int(exp) {
				seen = true
				break
			}
		}

		if !seen {
			results <- &Result{
				Beresp:        beresp,
				Err:           errors.UnexpectedStatus.With(err).Status(http.StatusBadGateway),
				RoundTripName: rtn,
			}
			return
		}
	}

	results <- &Result{
		Beresp:        beresp,
		Err:           err,
		RoundTripName: rtn,
	}
}
