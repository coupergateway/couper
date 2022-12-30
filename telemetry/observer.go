package telemetry

import (
	"context"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
)

func newBackendsObserver(memStore *cache.MemoryStore) error {
	bs := memStore.GetAllWithPrefix("backend_")
	var backends []interface{ Value() cty.Value }
	for _, b := range bs {
		if backend, ok := b.(interface{ Value() cty.Value }); ok {
			backends = append(backends, backend)
		}
	}

	meter := provider.Meter(instrumentation.BackendInstrumentationName)
	gauge, _ := meter.AsyncInt64().
		Gauge(
			instrumentation.BackendHealthState,
			instrument.WithDescription(string(unit.Dimensionless)),
		)

	onObserverFn := func(ctx context.Context) {
		backendsObserver(ctx, gauge, backends)
	}

	return meter.RegisterCallback([]instrument.Asynchronous{gauge}, onObserverFn)
}

func backendsObserver(ctx context.Context, gauge asyncint64.Gauge, backends []interface{ Value() cty.Value }) {
	for _, backend := range backends {
		v := backend.Value().AsValueMap()
		attrs := []attribute.KeyValue{
			attribute.String("backend_name", v["name"].AsString()),
			attribute.String("hostname", v["hostname"].AsString()),
			attribute.String("origin", v["origin"].AsString()),
		}
		var value int64 = 1 // default healthy due to anonymous ones
		health := v["health"].AsValueMap()
		if health["healthy"].False() {
			value = 0
		}

		gauge.Observe(ctx, value, attrs...)
	}
}
