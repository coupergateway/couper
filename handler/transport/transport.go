package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/connection"
	coupertls "github.com/avenga/couper/connection/tls"
	"github.com/avenga/couper/handler/ratelimit"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
)

// Config represents the transport <Config> object.
type Config struct {
	BackendName            string
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

	// TLS settings
	// Certificate is passed to all backends from the related cli option.
	Certificate []byte
	// CACertificate contains a per backend configured one.
	CACertificate tls.Certificate
	// ClientCertificate holds the one the backend will send during tls handshake if required.
	ClientCertificate tls.Certificate

	// Dynamic values
	Hostname string
	Origin   string
	Scheme   string
}

// NewTransport creates a new <*http.Transport> object by the given <*Config>.
func NewTransport(conf *Config, log *logrus.Entry) *http.Transport {
	tlsConf := coupertls.DefaultTLSConfig()
	if len(conf.Certificate) > 0 {
		tlsConf.RootCAs.AppendCertsFromPEM(conf.Certificate)
	}
	if conf.CACertificate.Leaf == nil {
		tlsConf.InsecureSkipVerify = conf.DisableCertValidation
	} else {
		tlsConf.RootCAs.AddCert(conf.CACertificate.Leaf)
	}

	if conf.ClientCertificate.Leaf != nil {
		clientCert := &conf.ClientCertificate
		tlsConf.GetClientCertificate = func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return clientCert, nil
		}
	}

	if conf.Origin != conf.Hostname {
		tlsConf.ServerName = conf.Hostname
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
		tlsConf.NextProtos = nil
	}

	logEntry := log.WithField("type", "couper_connection")

	transport := &http.Transport{
		DialContext: connection.NewDialContextFunc(&connection.Configuration{
			ContextName: conf.BackendName,
			ContextType: "backend",
			Hostname:    conf.Hostname,
			Origin:      conf.Origin,
		}, proxyFunc == nil, logEntry),
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
		logger.WithError(err).Warning("using default timing of ", target, " because an error occurred")
	}
	if src != "" && err != nil {
		return
	}
	*target = d
}
