package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/handler/ratelimit"
	"github.com/avenga/couper/telemetry"
	"golang.org/x/net/http/httpproxy"
)

// Config represents the transport <Config> object.
type Config struct {
	BackendName            string
	Certificate            []byte
	DisableCertValidation  bool
	DisableConnectionReuse bool
	HTTP2                  bool
	MaxConnections         int
	NoProxyFromEnv         bool
	Proxy                  string
	RateLimits             ratelimit.RateLimits

	ConnectTimeout time.Duration
	TTFBTimeout    time.Duration
	Timeout        time.Duration

	// Dynamic values
	Hostname string
	Origin   string
	Scheme   string
}

// NewTransport creates a new <*http.Transport> object by the given <*Config>.
func NewTransport(conf *Config, log *logrus.Entry) *http.Transport {
	certPool, err := x509.SystemCertPool()
	if err == nil {
		certPool.AppendCertsFromPEM(conf.Certificate)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: conf.DisableCertValidation,
		RootCAs:            certPool,
	}

	if conf.Origin != conf.Hostname {
		tlsConf.ServerName = conf.Hostname
	}

	d := &net.Dialer{
		KeepAlive: 60 * time.Second,
	}

	var proxyFunc func(req *http.Request) (*url.URL, error)
	if conf.Proxy != "" {
		proxyFunc = func(req *http.Request) (*url.URL, error) {
			proxyConf := &httpproxy.Config{
				HTTPProxy:  conf.Proxy,
				HTTPSProxy: conf.Proxy,
			}

			return proxyConf.ProxyFunc()(req.URL)
		}
	} else if !conf.NoProxyFromEnv {
		proxyFunc = http.ProxyFromEnvironment
	}

	// This is the documented way to disable http2. However, if a custom tls.Config or
	// DialContext is used h2 will also be disabled. To enable h2 the transport must be
	// explicitly configured, this can be done with the 'ForceAttemptHTTP2' below.
	var nextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper
	if !conf.HTTP2 {
		nextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}

	logEntry := log.WithField("type", "couper_connection")

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			address := addr
			if proxyFunc == nil {
				address = conf.Origin
			} // Otherwise, proxy connect will use this dial method and addr could be a proxy one.

			stx, span := telemetry.NewSpanFromContext(ctx, "connect", trace.WithAttributes(attribute.String("couper.address", addr)))
			defer span.End()

			connectTimeout, _ := ctx.Value(request.ConnectTimeout).(time.Duration)
			if connectTimeout > 0 {
				dtx, cancel := context.WithDeadline(stx, time.Now().Add(connectTimeout))
				stx = dtx
				defer cancel()
			}

			conn, cerr := d.DialContext(stx, network, address)
			if cerr != nil {
				return nil, fmt.Errorf("connecting to %s %q failed: %w", conf.BackendName, conf.Origin, cerr)
			}
			return NewOriginConn(stx, conn, conf, logEntry), nil
		},
		DisableCompression: true,
		DisableKeepAlives:  conf.DisableConnectionReuse,
		ForceAttemptHTTP2:  conf.HTTP2,
		MaxConnsPerHost:    conf.MaxConnections,
		Proxy:              proxyFunc,
		TLSClientConfig:    tlsConf,
		TLSNextProto:       nextProto,
	}

	return transport
}

func (c *Config) WithTarget(scheme, origin, hostname, proxyURL string) *Config {
	const defaultScheme = "http"
	conf := *c
	if scheme != "" {
		conf.Scheme = scheme
	} else {
		conf.Scheme = defaultScheme
		if conf.HTTP2 {
			conf.Scheme += "s"
		}
	}

	conf.Origin = origin
	conf.Hostname = hostname

	// Port required by transport.DialContext
	_, p, _ := net.SplitHostPort(origin)
	if p == "" {
		const port, tlsPort = "80", "443"
		if conf.Scheme == defaultScheme {
			conf.Origin += ":" + port
		} else {
			conf.Origin += ":" + tlsPort
		}
	}

	if proxyURL != "" {
		conf.Proxy = proxyURL
	}

	return &conf
}

func (c *Config) WithTimings(connect, ttfb, timeout string, logger *logrus.Entry) *Config {
	conf := *c
	parseDuration(connect, &conf.ConnectTimeout, "connect_timeout", logger)
	parseDuration(ttfb, &conf.TTFBTimeout, "ttfb_timeout", logger)
	parseDuration(timeout, &conf.Timeout, "timeout", logger)
	return &conf
}

// parseDuration sets the target value if the given duration string is not empty.
func parseDuration(src string, target *time.Duration, attributeName string, logger *logrus.Entry) {
	d, err := config.ParseDuration(attributeName, src, *target)
	if err != nil {
		logger.WithError(err).Warning("using default timing of ", target, " because an error occured")
	}
	if src != "" && err != nil {
		return
	}
	*target = d
}
