package middleware

import (
	"net/http"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

type MetricsHandler struct {
	durationBoundaries []float64
	handler            http.Handler
}

// NewMetricsHandler falls back to instrumentation.DefaultDurationSecondsBoundaries
// when durationBoundaries is empty.
func NewMetricsHandler(durationBoundaries []float64) Next {
	boundaries := instrumentation.DefaultDurationSecondsBoundaries
	if len(durationBoundaries) > 0 {
		boundaries = slices.Sorted(slices.Values(durationBoundaries))
	}

	return func(handler http.Handler) *NextHandler {
		return NewHandler(&MetricsHandler{
			durationBoundaries: boundaries,
			handler:            handler,
		}, handler)
	}
}

func (mh *MetricsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()
	mh.handler.ServeHTTP(rw, req)
	end := time.Since(start)

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
		metric.WithExplicitBucketBoundaries(mh.durationBoundaries...))

	option := metric.WithAttributes(metricsAttrs...)
	counter.Add(req.Context(), 1, option)
	duration.Record(req.Context(), end.Seconds(), option)
}
