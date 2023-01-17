package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
)

type MetricsHandler struct {
	handler http.Handler
}

func NewMetricsHandler() Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(&MetricsHandler{
			handler: handler,
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

	counter, _ := meter.SyncInt64().
		Counter(instrumentation.ClientRequest,
			instrument.WithDescription(string(unit.Dimensionless)))
	duration, _ := meter.SyncFloat64().
		Histogram(instrumentation.ClientRequestDuration,
			instrument.WithDescription(string(unit.Dimensionless)))

	counter.Add(req.Context(), 1, metricsAttrs...)
	duration.Record(req.Context(), end.Seconds(), metricsAttrs...)
}
