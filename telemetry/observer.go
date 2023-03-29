package telemetry

import (
	"context"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
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
	gauge, _ := meter.Int64ObservableGauge(
		instrumentation.BackendHealthState,
		instrument.WithDescription(string(unit.Dimensionless)),
	)

	onObserverFn := func(_ context.Context, observer metric.Observer) error {
		return backendsObserver(gauge, observer, backends)
	}

	_, err := meter.RegisterCallback(onObserverFn, gauge)
	return err
}

func backendsObserver(gauge instrument.Int64Observable, observer metric.Observer, backends []interface{ Value() cty.Value }) error {
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

		observer.ObserveInt64(gauge, value, attrs...)
	}
	return nil
}
