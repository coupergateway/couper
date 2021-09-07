package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const InstrumentationName = "github.com/avenga/couper/telemetry"

func NewSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	rootSpan := trace.SpanFromContext(ctx)
	return rootSpan.TracerProvider().
		Tracer(InstrumentationName).Start(ctx, name, opts...)
}
