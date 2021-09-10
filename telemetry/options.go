package telemetry

import "time"

type Options struct {
	Metrics              bool
	MetricsCollectPeriod time.Duration // internal collect for pusher configuration
	MetricsEndpoint      string
	MetricsExporter      string
	MetricsPort          int
	Traces               bool
	TracesEndpoint       string
}
