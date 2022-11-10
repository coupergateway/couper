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
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
)

const (
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
	var reqID string
	if uid, ok := ctx.Value(request.UID).(string); ok {
		reqID = uid
	}

	o := &OriginConn{
		Conn:         conn,
		conf:         conf,
		createdAt:    time.Now(),
		initialReqID: reqID,
		labels: []attribute.KeyValue{
			attribute.String("origin", conf.Origin),
			attribute.String("host", conf.Hostname),
			attribute.String("backend", conf.BackendName),
		},
		log:      entry,
		tlsState: nil,
	}

	if tlsConn, ok := conn.(*tls.Conn); ok {
		state := tlsConn.ConnectionState()
		o.tlsState = &state
	}
	entry.WithFields(o.logFields(eventOpen)).Debug()

	meter := provider.Meter("couper/connection")
	
	counter := metric.Must(meter).NewInt64Counter(instrumentation.BackendConnectionsTotal, metric.WithDescription(string(unit.Dimensionless)))
	gauge := metric.Must(meter).NewFloat64UpDownCounter(instrumentation.BackendConnections, metric.WithDescription(string(unit.Dimensionless)))
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

	if event == eventClose {
		since := time.Since(o.createdAt)

		meter := provider.Meter("couper/connection")
		duration := metric.Must(meter).
			NewFloat64Histogram(instrumentation.BackendConnectionsLifetime, metric.WithDescription(string(unit.Dimensionless)))
		meter.RecordBatch(context.Background(), o.labels, duration.Measurement(since.Seconds()))
		fields["lifetime"] = since.Milliseconds()
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

	meter := provider.Meter("couper/connection")
	gauge := metric.Must(meter).NewFloat64UpDownCounter(instrumentation.BackendConnections)
	meter.RecordBatch(context.Background(), o.labels,
		gauge.Measurement(-1),
	)
	return o.Conn.Close()
}
