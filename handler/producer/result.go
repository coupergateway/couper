package producer

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

// Result represents the producer <Result> object.
type Result struct {
	Beresp        *http.Response
	Err           error
	RoundTripName string
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

func roundtrip(rt http.RoundTripper, req *http.Request) *Result {
	rtn := req.Context().Value(request.RoundTripName).(string)
	span := trace.SpanFromContext(req.Context())

	beresp, err := rt.RoundTrip(req)
	span.End()

	if _, ok := err.(*errors.Error); ok {
		return &Result{
			Beresp:        beresp,
			Err:           err,
			RoundTripName: rtn,
		}
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
			return &Result{
				Beresp:        beresp,
				Err:           errors.UnexpectedStatus.With(err),
				RoundTripName: rtn,
			}
		}
	}

	return &Result{
		Beresp:        beresp,
		Err:           err,
		RoundTripName: rtn,
	}
}
