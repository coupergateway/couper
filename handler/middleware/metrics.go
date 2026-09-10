package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

type MetricsHandler struct {
	handler http.Handler
	clock   func() time.Time
}

func NewMetricsHandler() Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(&MetricsHandler{
			handler: handler,
			clock:   time.Now,
		}, handler)
	}
}

func (mh *MetricsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	start := mh.clock()
	mh.handler.ServeHTTP(rw, req)
	elapsed := mh.clock().Sub(start)

	metricsAttrs := []attribute.KeyValue{
		attribute.String("host", req.Host),
		attribute.String("method", req.Method),
	}

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		metricsAttrs = append(metricsAttrs, attribute.Int("code", rsw.StatusCode()))
	}

	meter := provider.Meter("couper/server")

	counter, _ := meter.Int64Counter(instrumentation.ClientRequest)
	duration, _ := meter.Float64Histogram(instrumentation.ClientRequestDuration,
		metric.WithExplicitBucketBoundaries(instrumentation.DefaultDurationSecondsBoundaries...))

	option := metric.WithAttributes(metricsAttrs...)
	counter.Add(req.Context(), 1, option)
	duration.Record(req.Context(), elapsed.Seconds(), option)
}
