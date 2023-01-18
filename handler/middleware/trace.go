package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/telemetry/instrumentation"
)

type TraceHandler struct {
	handler http.Handler
}

func NewTraceHandler() Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(&TraceHandler{
			handler: handler,
		}, handler)
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
	th.handler.ServeHTTP(rw, req)

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(rsw.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rsw.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}
}
