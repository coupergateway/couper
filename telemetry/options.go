package telemetry

import "time"

type Options struct {
	AgentAddr     string
	CollectPeriod time.Duration
	Exporter      string
	Metrics       bool
	ScrapePort    string
	ServiceName   string
	Traces        bool
}
