package handler

import (
	"context"
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

// TransportConfig represents the <TransportConfig> object.
type TransportConfig struct {
	BackendName            string
	ConnectTimeout         time.Duration
	DisableCertValidation  bool
	DisableConnectionReuse bool
	Hash                   string
	Hostname               string
	HTTP2                  bool
	MaxConnections         int
	NoProxyFromEnv         bool
	Origin                 string
	Proxy                  string
	Scheme                 string
	TTFBTimeout            time.Duration
	Timeout                time.Duration
}

func getTransport(conf *TransportConfig) *http.Transport {
	key := conf.Scheme + "|" + conf.Origin + "|" + conf.Hostname + "|" + conf.Hash

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
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s %q failed: %w", conf.BackendName, addr, err)
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
