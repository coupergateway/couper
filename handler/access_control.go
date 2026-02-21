package handler

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/coupergateway/couper/accesscontrol"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/server/writer"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

var (
	_ http.Handler                   = &AccessControl{}
	_ accesscontrol.ProtectedHandler = &AccessControl{}
)

type AccessControl struct {
	acl       accesscontrol.List
	protected http.Handler
}

func NewAccessControl(protected http.Handler, list accesscontrol.List) *AccessControl {
	return &AccessControl{
		acl:       list,
		protected: protected,
	}
}

func (a *AccessControl) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r, ok := rw.(*writer.Response)

	meter := provider.Meter(instrumentation.AccessControlInstrumentationName)
	counter, _ := meter.Int64Counter(instrumentation.AccessControlTotal)
	duration, _ := meter.Float64Histogram(instrumentation.AccessControlDuration)
	rateLimitedCounter, _ := meter.Int64Counter(instrumentation.AccessControlRateLimited)

	for _, control := range a.acl {
		if ok && !control.DisablePrivateCaching() {
			r.AddPrivateCC()
		}

		start := time.Now()
		err := control.Validate(req)
		elapsed := time.Since(start).Seconds()

		status := "granted"
		if err != nil {
			status = "denied"
		}

		acName := control.Label()
		acType := control.Kind()

		attrs := metric.WithAttributes(
			attribute.String("ac_name", acName),
			attribute.String("ac_type", acType),
			attribute.String("status", status),
		)
		counter.Add(req.Context(), 1, attrs)

		durationAttrs := metric.WithAttributes(
			attribute.String("ac_name", acName),
			attribute.String("ac_type", acType),
		)
		duration.Record(req.Context(), elapsed, durationAttrs)

		if acType == "rate_limiter" && err != nil {
			acNameAttr := metric.WithAttributes(attribute.String("ac_name", acName))
			rateLimitedCounter.Add(req.Context(), 1, acNameAttr)
		}

		if err != nil {
			*req = *req.WithContext(context.WithValue(req.Context(), request.Error, err))
			control.ErrorHandler().ServeHTTP(rw, req)
			return
		}
	}
	a.protected.ServeHTTP(rw, req)
}

func (a *AccessControl) Child() http.Handler {
	return a.protected
}

func (a *AccessControl) String() string {
	if h, ok := a.protected.(interface{ String() string }); ok {
		return h.String()
	}
	return "AccessControl"
}
