package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/telemetry/instrumentation"
)

func NewSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	rootSpan := trace.SpanFromContext(ctx)
	return rootSpan.TracerProvider().
		Tracer(instrumentation.Name).Start(ctx, name, opts...)
}
