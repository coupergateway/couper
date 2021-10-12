package telemetry

import "time"

type Options struct {
	Metrics              bool
	MetricsCollectPeriod time.Duration // internal collect for pusher configuration
	MetricsEndpoint      string
	MetricsExporter      string
	MetricsPort          int
	ServiceName          string
	Traces               bool
	TracesEndpoint       string
}
