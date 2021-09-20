package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
)

type TraceHandler struct {
	handler http.Handler
}

func NewTraceHandler() Next {
	return func(handler http.Handler) http.Handler {
		return &TraceHandler{
			handler: handler,
		}
	}
}

func (th *TraceHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	spanName := req.URL.EscapedPath()
	opts := []trace.SpanStartOption{
		trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", req)...),
		trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(req)...),
		trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest("couper", spanName, req)...),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("couper.uid", req.Context().Value(request.UID).(string))),
	}

	tracer := otel.GetTracerProvider().Tracer(instrumentation.Name)
	ctx, span := tracer.Start(req.Context(), spanName, opts...)
	defer span.End()

	*req = *req.WithContext(ctx)
	start := time.Now()
	th.handler.ServeHTTP(rw, req)
	end := time.Since(start)

	metricsAttrs := []attribute.KeyValue{
		attribute.String("host", req.Host),
		attribute.String("method", req.Method),
	}

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(rsw.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rsw.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)

		metricsAttrs = append(metricsAttrs, attribute.Int("code", rsw.StatusCode()))
	}

	meter := provider.Meter("couper/server")
	counter := metric.Must(meter).NewInt64Counter(instrumentation.ClientRequest, metric.WithDescription(string(unit.Dimensionless)))
	duration := metric.Must(meter).
		NewFloat64Histogram(instrumentation.ClientRequestDuration, metric.WithDescription(string(unit.Dimensionless)))
	meter.RecordBatch(req.Context(), metricsAttrs,
		counter.Measurement(1),
		duration.Measurement(end.Seconds()),
	)
}
