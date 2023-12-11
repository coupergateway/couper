package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/telemetry/instrumentation"
)

type TraceHandler struct {
	handler     http.Handler
	parentOnly  bool
	trustParent bool
}

func NewTraceHandler(parentOnly, trustParent bool) Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(&TraceHandler{
			handler:     handler,
			parentOnly:  parentOnly,
			trustParent: trustParent,
		}, handler)
	}
}

func (th *TraceHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	traceParent := req.Header.Get("Traceparent")
	if th.parentOnly && traceParent == "" {
		// Only trace if a 'traceparent' header is present.
		// This allows e.g. an ingress to trace based on percentage configuration.
		th.handler.ServeHTTP(rw, req)
		return
	}

	spanName := req.URL.EscapedPath()
	opts := []trace.SpanStartOption{
		trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", req)...),
		trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(req)...),
		trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest("couper", spanName, req)...),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("couper.uid", req.Context().Value(request.UID).(string))),
	}

	parentCtx := req.Context()
	if th.trustParent {
		parentCtx = otel.GetTextMapPropagator().Extract(parentCtx, propagation.HeaderCarrier(req.Header))
	}

	tracer := otel.GetTracerProvider().Tracer(instrumentation.Name)
	ctx, span := tracer.Start(parentCtx, spanName, opts...)
	defer span.End()

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	*req = *req.WithContext(ctx)
	th.handler.ServeHTTP(rw, req)

	if rsw, ok := rw.(logging.RecorderInfo); ok {
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(rsw.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(rsw.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}
}
