package transport

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"

	"github.com/avenga/couper/config/request"
)

const (
	MetricConnections              = "connections_count"
	MetricConnectionsTotal         = "connections_total_count"
	MetricConnectionsLifetimeTotal = "connections_lifetime_total"

	eventOpen  = "open"
	eventClose = "close"
)

// OriginConn wraps the original net.Conn created by net.DialContext or transport.DialTLS for debug purposes.
type OriginConn struct {
	net.Conn

	connClosedMu sync.Mutex
	connClosed   bool

	conf *Config

	createdAt    time.Time
	initialReqID string
	labels       []attribute.KeyValue
	log          *logrus.Entry
	tlsState     *tls.ConnectionState
}

// NewOriginConn creates a new wrapper with logging context.
func NewOriginConn(ctx context.Context, conn net.Conn, conf *Config, entry *logrus.Entry) *OriginConn {
	backendName := "default"
	if conf.BackendName != "" {
		backendName = conf.BackendName
	}
	o := &OriginConn{
		Conn:         conn,
		conf:         conf,
		createdAt:    time.Now(),
		initialReqID: ctx.Value(request.UID).(string),
		labels: []attribute.KeyValue{
			attribute.String("origin", conf.Origin),
			attribute.String("host", conf.Hostname),
			attribute.String("backend", backendName),
		},
		log:      entry,
		tlsState: nil,
	}

	if tlsConn, ok := conn.(*tls.Conn); ok {
		state := tlsConn.ConnectionState()
		o.tlsState = &state
	}
	entry.WithFields(o.logFields(eventOpen)).Debug()

	meter := global.Meter("couper/connection")

	counter := metric.Must(meter).NewInt64Counter(MetricConnectionsTotal)
	gauge := metric.Must(meter).NewFloat64UpDownCounter(MetricConnections)
	meter.RecordBatch(ctx, o.labels,
		counter.Measurement(1),
		gauge.Measurement(1),
	)
	return o
}

func (o *OriginConn) logFields(event string) logrus.Fields {
	fields := logrus.Fields{
		"event":       event,
		"initial_uid": o.initialReqID,
		"localAddr":   o.LocalAddr().String(),
		"origin":      o.conf.Origin,
		"remoteAddr":  o.RemoteAddr().String(),
	}

	var d time.Duration
	if event == eventClose {
		d = time.Now().Sub(o.createdAt) / time.Millisecond

		meter := global.Meter("couper/connection")
		counter := metric.Must(meter).NewFloat64Counter(MetricConnectionsLifetimeTotal)
		meter.RecordBatch(context.Background(), o.labels, counter.Measurement(float64(d)))
		fields["lifetime"] = d
	}
	return logrus.Fields{
		"connection": fields,
	}
}

func (o *OriginConn) Close() error {
	o.connClosedMu.Lock()
	if o.connClosed {
		o.connClosedMu.Unlock()
		return nil
	}
	o.connClosed = true
	o.connClosedMu.Unlock()

	o.log.WithFields(o.logFields(eventClose)).Debug()

	meter := global.Meter("couper/connection")
	gauge := metric.Must(meter).NewFloat64UpDownCounter(MetricConnections)
	meter.RecordBatch(context.Background(), o.labels,
		gauge.Measurement(-1),
	)
	return o.Conn.Close()
}
