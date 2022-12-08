package connection

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/telemetry"
)

// Dialer is used for every new Conn. Replace on init() for other settings.
var Dialer = &net.Dialer{Timeout: time.Second * 60}

func NewDialContextFunc(conf *Configuration, replaceAddr bool, log *logrus.Entry) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		address := addr
		// replaceAddr is useful in combination with http.Transport.
		// Otherwise, proxy connect will use this dial method and addr could be a proxy one.
		if replaceAddr {
			address = conf.Origin
		}

		stx, span := telemetry.NewSpanFromContext(ctx, "connect", trace.WithAttributes(attribute.String("couper.address", addr)))
		defer span.End()

		connectTimeout, _ := ctx.Value(request.ConnectTimeout).(time.Duration)
		if connectTimeout > 0 {
			dtx, cancel := context.WithDeadline(stx, time.Now().Add(connectTimeout))
			stx = dtx
			defer cancel()
		}

		conn, cerr := Dialer.DialContext(stx, network, address)
		if cerr != nil {
			host, port, _ := net.SplitHostPort(conf.Origin)
			if port != "80" && port != "443" {
				host = conf.Origin
			}
			if os.IsTimeout(cerr) || cerr == context.DeadlineExceeded {
				return nil, fmt.Errorf("connecting to %s '%s' failed: i/o timeout", conf.ContextType, host)
			}
			return nil, fmt.Errorf("connecting to %s '%s' failed: %w", conf.ContextType, conf.Origin, cerr)
		}
		return NewConn(stx, conn, conf, log), nil
	}
}
