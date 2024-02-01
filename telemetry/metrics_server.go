package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/telemetry/handler"
)

type MetricsServer struct {
	log    *logrus.Entry
	server *http.Server
}

func NewMetricsServer(log *logrus.Entry, registry *WrappedRegistry, port int) *MetricsServer {
	server := &http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: handler.NewWrappedHandler(log, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
			ErrorLog:          log,
			Registry:          registry,
			Timeout:           time.Second * 2,
		})),
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
