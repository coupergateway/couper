package provider

import (
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var meterProvider metric.MeterProvider
var meterMu sync.RWMutex

func init() {
	meterMu.Lock()
	defer meterMu.Unlock()
	meterProvider = global.MeterProvider() // defaults to noop
}

func SetMeterProvider(provider metric.MeterProvider) {
	meterMu.Lock()
	defer meterMu.Unlock()
	meterProvider = provider
}

func Meter(instrumentationName string) metric.Meter {
	meterMu.RLock()
	defer meterMu.RUnlock()
	return meterProvider.Meter(instrumentationName)
}
