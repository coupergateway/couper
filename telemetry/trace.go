package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/utils"
)

var InstrumentationName = "github.com/avenga/couper/telemetry"
var InstrumentationVersion = utils.VersionName

func NewSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	rootSpan := trace.SpanFromContext(ctx)
	return rootSpan.TracerProvider().
		Tracer(
			InstrumentationName,
			trace.WithInstrumentationVersion(InstrumentationVersion),
		).Start(ctx, name, opts...)
}
