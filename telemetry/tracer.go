package telemetry

import (
	"net/http"

	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/utils"
)

var InstrumentationName = "github.com/avenga/couper/telemetry"
var InstrumentationVersion = utils.VersionName

type TraceHandler struct {
	service string
	tracer  trace.Tracer
	handler http.Handler
}

func NewTraceHandler(service string) func(http.Handler) http.Handler {
	traceProvider := otel.GetTracerProvider()
	tracer := traceProvider.Tracer(
		InstrumentationName,
		trace.WithInstrumentationVersion(InstrumentationVersion),
	)

	return func(handler http.Handler) http.Handler {
		return &TraceHandler{
			service: service,
			tracer:  tracer,
			handler: handler,
		}
	}
}

func (th *TraceHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	spanName := req.URL.EscapedPath()
	opts := []trace.SpanStartOption{
		trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", req)...),
		trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(req)...),
		trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(th.service, spanName, req)...),
		trace.WithSpanKind(trace.SpanKindServer),
	}
	span := trace.SpanFromContext(req.Context())
	span.SetName(spanName)
	ctx, span := th.tracer.Start(req.Context(), spanName, opts...)
	defer span.End()

	th.handler.ServeHTTP(rw, req.WithContext(ctx))

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(rsw.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rsw.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}

}
