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

		d := &net.Dialer{Timeout: conf.ConnectTimeout}

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

		var nextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper
		if !conf.HTTP2 {
			nextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s %q failed: %w", conf.BackendName, addr, err)
				}
				return conn, nil
			},
			Dial: (&net.Dialer{
				KeepAlive: 60 * time.Second,
			}).Dial,
			DisableCompression:    true,
			DisableKeepAlives:     conf.DisableConnectionReuse,
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

func (c *Config) With(scheme, origin, hostname string) *Config {
	conf := *c
	conf.Scheme = scheme
	conf.Origin = origin
	conf.Hostname = hostname
	return &conf
}

func (c *Config) hash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", c)))
	return fmt.Sprintf("%x", h.Sum(nil))
}
