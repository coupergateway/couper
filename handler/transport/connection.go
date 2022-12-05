package transport

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
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

	counter, gauge := newMeterCounter()

	counter.Add(ctx, 1, o.labels...)
	gauge.Add(ctx, 1, o.labels...)

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
		duration, _ := meter.SyncFloat64().Histogram(
			instrumentation.BackendConnectionsLifetime,
			instrument.WithDescription(string(unit.Dimensionless)),
		)
		duration.Record(context.Background(), since.Seconds(), o.labels...)

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

	_, gauge := newMeterCounter()
	gauge.Add(context.Background(), -1, o.labels...)

	return o.Conn.Close()
}

func newMeterCounter() (syncint64.Counter, syncfloat64.UpDownCounter) {
	meter := provider.Meter("couper/connection")

	counter, _ := meter.SyncInt64().
		Counter(instrumentation.BackendConnectionsTotal, instrument.WithDescription(string(unit.Dimensionless)))
	gauge, _ := meter.SyncFloat64().UpDownCounter(
		instrumentation.BackendConnections,
		instrument.WithDescription(string(unit.Dimensionless)),
	)
	return counter, gauge
}
