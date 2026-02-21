package telemetry

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/config/request"
)

type capturingRT struct {
	lastReq  *http.Request
	response *http.Response
	err      error
}

func (c *capturingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.lastReq = req
	return c.response, c.err
}

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

var traceparentRE = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

// newRequestWithParentSpan creates an HTTP request with a parent SERVER span
// in the context. This is needed because NewSpanFromContext retrieves the tracer
// from the parent span's TracerProvider.
func newRequestWithParentSpan(t *testing.T, url string, backendName string) (*http.Request, trace.Span) {
	t.Helper()
	ctx := context.Background()
	if backendName != "" {
		ctx = context.WithValue(ctx, request.BackendName, backendName)
	}
	tracer := otel.GetTracerProvider().Tracer("test")
	ctx, parentSpan := tracer.Start(ctx, "server-request", trace.WithSpanKind(trace.SpanKindServer))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req, parentSpan
}

func findSpan(spans tracetest.SpanStubs, name string) *tracetest.SpanStub {
	for i, s := range spans {
		if s.Name == name {
			return &spans[i]
		}
	}
	return nil
}

func spanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}
	return names
}

func TestInstrumentedRoundTripper_InjectsTraceparent(t *testing.T) {
	setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	tp := inner.lastReq.Header.Get("Traceparent")
	if tp == "" {
		t.Fatal("expected Traceparent header to be injected, got empty")
	}
	if !traceparentRE.MatchString(tp) {
		t.Errorf("Traceparent header has invalid format: %q", tp)
	}
}

func TestInstrumentedRoundTripper_SpanNameWithBackendName(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatalf("expected span 'backend.mybackend', got: %v", spanNames(exporter.GetSpans()))
	}
}

func TestInstrumentedRoundTripper_SpanNameFallback(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend")
	if span == nil {
		t.Fatalf("expected span 'backend', got: %v", spanNames(exporter.GetSpans()))
	}
}

func TestInstrumentedRoundTripper_SpanKindClient(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("span not found")
	}
	if span.SpanKind != trace.SpanKindClient {
		t.Errorf("expected SpanKindClient, got %v", span.SpanKind)
	}
}

func TestInstrumentedRoundTripper_OriginAttribute(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com:8080/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("span not found")
	}
	var found bool
	for _, attr := range span.Attributes {
		if string(attr.Key) == "couper.origin" && attr.Value.AsString() == "example.com:8080" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected couper.origin=example.com:8080, got attributes: %v", span.Attributes)
	}
}

func TestInstrumentedRoundTripper_ResponseStatus200(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("span not found")
	}
	if span.Status.Code != codes.Unset {
		t.Errorf("expected status Unset for 200, got %v", span.Status.Code)
	}
}

func TestInstrumentedRoundTripper_ResponseStatus500(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 500, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("span not found")
	}
	if span.Status.Code != codes.Error {
		t.Errorf("expected status Error for 500, got %v", span.Status.Code)
	}
}

func TestInstrumentedRoundTripper_SpanEndedOnError(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{err: errors.New("connection refused")}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error")
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("expected span to be recorded even on error")
	}
	if span.EndTime.IsZero() {
		t.Error("expected span EndTime to be set")
	}
}

func TestInstrumentedRoundTripper_RequestResponseEvents(t *testing.T) {
	exporter := setupTraceProvider(t)
	inner := &capturingRT{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
	}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	span := findSpan(exporter.GetSpans(), "backend.mybackend")
	if span == nil {
		t.Fatal("span not found")
	}

	eventNames := make([]string, len(span.Events))
	for i, e := range span.Events {
		eventNames[i] = e.Name
	}
	for _, want := range []string{"backend.mybackend.request", "backend.mybackend.response"} {
		found := false
		for _, name := range eventNames {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected event %q, got: %v", want, eventNames)
		}
	}
}

func TestInstrumentedRoundTripper_ErrorPropagated(t *testing.T) {
	setupTraceProvider(t)
	wantErr := errors.New("connection refused")
	inner := &capturingRT{err: wantErr}
	rt := NewInstrumentedRoundTripper(inner)
	req, parentSpan := newRequestWithParentSpan(t, "http://example.com/api", "mybackend")
	defer parentSpan.End()

	_, gotErr := rt.RoundTrip(req)
	if gotErr != wantErr {
		t.Errorf("expected error %v, got %v", wantErr, gotErr)
	}
}
