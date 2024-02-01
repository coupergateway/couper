package telemetry

import (
	"context"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/telemetry/instrumentation"
	"github.com/coupergateway/couper/telemetry/provider"
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
	gauge, _ := meter.Int64ObservableGauge(instrumentation.BackendHealthState)

	onObserverFn := func(_ context.Context, observer metric.Observer) error {
		return backendsObserver(gauge, observer, backends)
	}

	_, err := meter.RegisterCallback(onObserverFn, gauge)
	return err
}

func backendsObserver(gauge metric.Int64Observable, observer metric.Observer, backends []interface{ Value() cty.Value }) error {
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

		option := metric.WithAttributes(attrs...)
		observer.ObserveInt64(gauge, value, option)
	}
	return nil
}
