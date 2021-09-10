package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/prometheus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/logging"
)

type MetricsServer struct {
	log    *logrus.Entry
	server *http.Server
}

func NewMetricsServer(log *logrus.Entry, exporter *prometheus.Exporter, port int) *MetricsServer {
	accessLog := logging.NewAccessLog(nil, log)

	uidHandler := middleware.NewUIDHandler(&config.DefaultSettings, "")(exporter)
	logHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		accessLog.ServeHTTP(rw, req, uidHandler, time.Now())
	})

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: logHandler,
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
	return m.server.Close()
}
