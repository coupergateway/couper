package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
)

type Metrics struct {
	exporter     metric.Exporter
	log          *logrus.Entry
	promExporter *prometheus.Exporter
	server       *http.Server
}

type Options struct {
	exporter string
	port     string
}

func NewMetrics(opts *Options, log *logrus.Entry) (*Metrics, error) {
	var exporter string
	if opts == nil || opts.exporter == "" {
		exporter = "prometheus"
	} else {
		exporter = opts.exporter
	}

	metrics := &Metrics{
		log: log.WithField("type", "couper_metrics"),
	}

	switch exporter {
	case "prometheus":
		promExporter, err := newPromExporter(log)
		if err != nil {
			return nil, err
		}
		metrics.promExporter = promExporter
	default:
		return nil, fmt.Errorf("metrics: unsupported exporter: %s", exporter)
	}

	global.SetMeterProvider(metrics.promExporter.MeterProvider())
	return metrics, nil
}

func (m *Metrics) ListenAndServe() {
	accessLog := logging.NewAccessLog(nil, m.log)
	serveHTTP := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r := writer.NewResponseWriter(rw, "")
		accessLog.ServeHTTP(r, req, m.promExporter, time.Now())
	})
	m.log.Info("couper is serving metrics: :9090")
	m.server = &http.Server{Addr: ":9090", Handler: serveHTTP}
	err := m.server.ListenAndServe()
	if err != nil {
		m.log.WithError(err).Error()
	}
}

func (m *Metrics) Close() error {
	if m.server == nil {
		return nil
	}
	return m.server.Close()
}
