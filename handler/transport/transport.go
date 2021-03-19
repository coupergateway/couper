package transport

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/http/httpproxy"
)

var transports sync.Map

// Config represents the transport <Config> object.
type Config struct {
	BackendName            string
	DisableCertValidation  bool
	DisableConnectionReuse bool
	HTTP2                  bool
	MaxConnections         int
	NoProxyFromEnv         bool
	Proxy                  string

	ConnectTimeout time.Duration
	TTFBTimeout    time.Duration
	Timeout        time.Duration

	// Dynamic values
	Hostname string
	Origin   string
	Scheme   string
}

// Get creates a new <*http.Transport> object by the given <*Config>.
func Get(conf *Config) *http.Transport {
	key := conf.hash()

	transport, ok := transports.Load(key)
	if !ok {
		tlsConf := &tls.Config{
			InsecureSkipVerify: conf.DisableCertValidation,
		}
		if conf.Origin != conf.Hostname {
			tlsConf.ServerName = conf.Hostname
		}

		d := &net.Dialer{
			KeepAlive: 60 * time.Second,
			Timeout:   conf.ConnectTimeout,
		}

		var proxyFunc func(req *http.Request) (*url.URL, error)
		if conf.Proxy != "" {
			proxyFunc = func(req *http.Request) (*url.URL, error) {
				config := &httpproxy.Config{
					HTTPProxy:  conf.Proxy,
					HTTPSProxy: conf.Proxy,
				}

				return config.ProxyFunc()(req.URL)
			}
		} else if !conf.NoProxyFromEnv {
			proxyFunc = http.ProxyFromEnvironment
		}

		// This is the documented way to disable http2. However if a custom tls.Config or
		// DialContext is used h2 will also be disabled. To enable h2 the transport must be
		// explicitly configured, this can be done with the 'ForceAttemptHTTP2' below.
		var nextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper
		if !conf.HTTP2 {
			nextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				address := addr
				if proxyFunc == nil {
					address = conf.Origin
				} // Otherwise proxy connect will use this dial method and addr could be a proxy one.
				conn, err := d.DialContext(ctx, network, address)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s %q failed: %w", conf.BackendName, conf.Origin, err)
				}
				return conn, nil
			},
			DisableCompression:    true,
			DisableKeepAlives:     conf.DisableConnectionReuse,
			ForceAttemptHTTP2:     conf.HTTP2,
			MaxConnsPerHost:       conf.MaxConnections,
			Proxy:                 proxyFunc,
			ResponseHeaderTimeout: conf.TTFBTimeout,
			TLSClientConfig:       tlsConf,
			TLSNextProto:          nextProto,
		}

		transports.Store(key, transport)
	}

	if t, ok := transport.(*http.Transport); ok {
		return t
	}

	return nil
}

func (c *Config) With(scheme, origin, hostname, proxyURL string) *Config {
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

func (c *Config) hash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", c)))
	return fmt.Sprintf("%x", h.Sum(nil))
}
