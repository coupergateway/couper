package telemetry

import (
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"

	"github.com/avenga/couper/telemetry/handler"
)

type MetricsServer struct {
	log    *logrus.Entry
	server *http.Server
}

func NewMetricsServer(log *logrus.Entry, exporter *otelprom.Exporter, port int) *MetricsServer {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handler.NewWrappedHandler(log, nil), // TODO: how to promhttp export
	}

	return &MetricsServer{
		log:    log,
		server: server,
	}
}

func (m *MetricsServer) ListenAndServe() {
	m.log.Infof("couper is serving metrics: %s", m.server.Addr)

	err := m.server.ListenAndServe()
	if err != nil {
		m.log.WithError(err).Error("serving metrics failed")
	}
}

func (m *MetricsServer) Close() error {
	if m == nil || m.server == nil {
		return nil
	}
	m.log.Infof("shutdown metrics server: %s", m.server.Addr)
	return m.server.Close()
}
