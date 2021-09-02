package telemetry

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/avenga/couper/logging"
)

type Metrics struct {
	exporter     metric.Exporter
	log          *logrus.Entry
	promExporter *prometheus.Exporter
	server       *http.Server
}

func (m *Metrics) ListenAndServe() {
	accessLog := logging.NewAccessLog(nil, m.log)
	serveHTTP := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		//r := writer.NewResponseWriter(rw, "")
		accessLog.ServeHTTP(rw, req, m.promExporter, time.Now())
	})
	m.log.Info("couper is serving metrics: :9090")
	m.server = &http.Server{
		Addr:    ":9090",
		Handler: serveHTTP,
	}
	err := m.server.ListenAndServe()
	if err != nil {
		m.log.WithError(err).Error()
	}
}

func (m *Metrics) Close() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Close()
}
