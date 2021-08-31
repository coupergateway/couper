package telemetry

import (
	"log"
	"net/http"

	"github.com/avenga/couper/logging"

	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/utils"
)

func init() {
	initTracer()
}

var tracerName = "github.com/avenga/couper/telemetry"

type TraceHandler struct {
	propagators propagation.TextMapPropagator
	service     string
	tracer      trace.Tracer
	handler     http.Handler
}

func NewTraceHandler(service string) func(http.Handler) http.Handler {
	traceProvider := otel.GetTracerProvider()
	tracer := traceProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion(utils.VersionName),
	)
	propagators := otel.GetTextMapPropagator()

	return func(handler http.Handler) http.Handler {
		return &TraceHandler{
			propagators: propagators,
			service:     service,
			tracer:      tracer,
			handler:     handler,
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

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}
