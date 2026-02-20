package telemetry

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
)

var _ http.RoundTripper = &InstrumentedRoundTripper{}

// InstrumentedRoundTripper wraps an http.RoundTripper with OpenTelemetry
// tracing and metrics. It creates a CLIENT span for each outgoing request,
// injects the trace context (traceparent) into request headers, records
// span attributes and metrics.
type InstrumentedRoundTripper struct {
	next http.RoundTripper
}

// NewInstrumentedRoundTripper wraps the given RoundTripper with tracing
// and metrics instrumentation.
func NewInstrumentedRoundTripper(next http.RoundTripper) *InstrumentedRoundTripper {
	return &InstrumentedRoundTripper{next: next}
}

func (t *InstrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	backendName, _ := ctx.Value(request.BackendName).(string)

	spanName := "backend"
	if backendName != "" {
		spanName += "." + backendName
	}

	ctx, span := NewSpanFromContext(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.HTTPClientAttributesFromHTTPRequest(req)...),
	)
	defer span.End()

	span.SetAttributes(KeyOrigin.String(req.URL.Host))

	// Inject trace context into outgoing request headers so the
	// backend receives the correct traceparent.
	if req.Header == nil {
		req.Header = http.Header{}
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	req = req.WithContext(ctx)

	// Metrics setup
	meter := provider.Meter(instrumentation.BackendInstrumentationName)
	counter, _ := meter.Int64Counter(instrumentation.BackendRequest)
	duration, _ := meter.Float64Histogram(instrumentation.BackendRequestDuration)

	attrs := []attribute.KeyValue{
		attribute.String("backend_name", backendName),
		attribute.String("hostname", req.Host),
		attribute.String("method", req.Method),
		attribute.String("origin", req.URL.Host),
	}

	start := time.Now()
	span.AddEvent(spanName + ".request")
	beresp, err := t.next.RoundTrip(req)
	span.AddEvent(spanName + ".response")
	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if beresp != nil {
		attrs = append(attrs, attribute.Key("code").Int(beresp.StatusCode))
		respAttrs := semconv.HTTPAttributesFromHTTPStatusCode(beresp.StatusCode)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(beresp.StatusCode)
		span.SetAttributes(respAttrs...)
		span.SetStatus(spanStatus, spanMessage)
	}

	option := metric.WithAttributes(attrs...)
	counter.Add(req.Context(), 1, option)
	duration.Record(req.Context(), elapsed, option)

	return beresp, err
}
