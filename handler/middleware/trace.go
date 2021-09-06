package middleware

import (
	"net/http"

	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/telemetry"
)

type TraceHandler struct {
	service string
	handler http.Handler
}

func NewTraceHandler(service string) Next {
	return func(handler http.Handler) http.Handler {
		return &TraceHandler{
			service: service,
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
		trace.WithAttributes(telemetry.KeyUID.String(req.Context().Value(request.UID).(string))),
	}

	ctx, span := telemetry.NewSpanFromContext(req.Context(), spanName, opts...)
	defer span.End()

	th.handler.ServeHTTP(rw, req.WithContext(ctx))

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(rsw.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rsw.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}

}
