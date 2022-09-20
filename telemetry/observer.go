package telemetry

import (
	"context"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
)

func newBackendsObserver(memStore cache.Storage) {
	bs := memStore.GetAllWithPrefix("backend_")
	var backends []interface{ Value() cty.Value }
	for _, b := range bs {
		if backend, ok := b.(interface{ Value() cty.Value }); ok {
			backends = append(backends, backend)
		}
	}

	onObserverFn := func(_ context.Context, result metric.Int64ObserverResult) {
		backendsObserver(backends, result)
	}

	meter := provider.Meter(instrumentation.BackendInstrumentationName)
	metric.Must(meter).
		NewInt64GaugeObserver(instrumentation.BackendHealthState, onObserverFn, metric.WithDescription(string(unit.Dimensionless)))
}

func backendsObserver(backends []interface{ Value() cty.Value }, result metric.Int64ObserverResult) {
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

		result.Observe(value, attrs...)
	}
}
