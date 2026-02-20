package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/server/writer"
)

func setupTraceProvider(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	return exporter
}

const (
	knownTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	knownTraceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
)

func newTestRequest(t *testing.T, traceparent string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	ctx := context.WithValue(req.Context(), request.UID, "test-uid-123")
	req = req.WithContext(ctx)
	if traceparent != "" {
		req.Header.Set("Traceparent", traceparent)
	}
	return req
}

func TestTraceHandler_TrustParentExtractsParentContext(t *testing.T) {
	exporter := setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(false, true)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, knownTraceparent)

	handler.ServeHTTP(rw, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.SpanContext.TraceID().String() != knownTraceID {
		t.Errorf("expected trace ID %s, got %s", knownTraceID, span.SpanContext.TraceID())
	}
	if span.Parent.TraceID().String() != knownTraceID {
		t.Errorf("expected parent trace ID %s, got %s", knownTraceID, span.Parent.TraceID())
	}
}

func TestTraceHandler_TrustParentFalseIgnoresTraceparent(t *testing.T) {
	exporter := setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(false, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, knownTraceparent)

	handler.ServeHTTP(rw, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.SpanContext.TraceID().String() == knownTraceID {
		t.Error("expected different trace ID when trustParent=false, but got the same as incoming header")
	}
}

func TestTraceHandler_ParentOnlySkipsWhenNoHeader(t *testing.T) {
	exporter := setupTraceProvider(t)

	var called bool
	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		called = true
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(true, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, "")

	handler.ServeHTTP(rw, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if len(exporter.GetSpans()) != 0 {
		t.Errorf("expected no spans when parentOnly and no Traceparent, got %d", len(exporter.GetSpans()))
	}
}

func TestTraceHandler_ParentOnlyTracesWhenHeaderPresent(t *testing.T) {
	exporter := setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(true, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, knownTraceparent)

	handler.ServeHTTP(rw, req)

	if len(exporter.GetSpans()) == 0 {
		t.Fatal("expected span when parentOnly and Traceparent is present")
	}
}

func TestTraceHandler_InjectsTraceparentIntoResponse(t *testing.T) {
	setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(false, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, "")

	handler.ServeHTTP(rw, req)

	tp := rec.Header().Get("Traceparent")
	if tp == "" {
		t.Fatal("expected Traceparent header in response")
	}
}

func TestTraceHandler_ResponseTraceparentMatchesSpan(t *testing.T) {
	exporter := setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	handler := NewTraceHandler(false, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, "")

	handler.ServeHTTP(rw, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	spanTraceID := spans[0].SpanContext.TraceID().String()
	tp := rec.Header().Get("Traceparent")
	parts := strings.Split(tp, "-")
	if len(parts) < 2 {
		t.Fatalf("invalid Traceparent format: %q", tp)
	}
	responseTraceID := parts[1]
	if responseTraceID != spanTraceID {
		t.Errorf("response Traceparent trace ID %q does not match span trace ID %q", responseTraceID, spanTraceID)
	}
}

func TestTraceHandler_SetsSpanStatusFromHTTPStatus(t *testing.T) {
	exporter := setupTraceProvider(t)

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	})
	handler := NewTraceHandler(false, false)(inner)

	rec := httptest.NewRecorder()
	rw := writer.NewResponseWriter(rec, "")
	req := newTestRequest(t, "")

	handler.ServeHTTP(rw, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("expected status Error for 500, got %v", spans[0].Status.Code)
	}
}
