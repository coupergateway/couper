package telemetry

import (
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/exporters/prometheus"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func newPromExporter(_ *logrus.Entry) (*prometheus.Exporter, error) {
	config := prometheus.Config{} // configured by otel; todo: promHTTPhandler opts/error logging can't be set this way
	ctlr := controller.New(
		processor.New(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
	)
	promExporter, err := prometheus.New(config, ctlr)
	return promExporter, err
}
