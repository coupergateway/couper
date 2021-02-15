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

type transportConfig struct {
	backendName            string
	connectTimeout         time.Duration
	disableCertValidation  bool
	disableConnectionReuse bool
	hash                   string
	hostname               string
	http2                  bool
	maxConnections         int
	noProxyFromEnv         bool
	origin                 string
	proxy                  string
	scheme                 string
	ttfbTimeout            time.Duration
	timeout                time.Duration
}

func getTransport(conf *transportConfig) *http.Transport {
	key := conf.scheme + "|" + conf.origin + "|" + conf.hostname + "|" + conf.hash

	transport, ok := transports.Load(key)
	if !ok {
		tlsConf := &tls.Config{
			InsecureSkipVerify: conf.disableCertValidation,
		}
		if conf.origin != conf.hostname {
			tlsConf.ServerName = conf.hostname
		}

		d := &net.Dialer{Timeout: conf.connectTimeout}

		var proxyFunc func(req *http.Request) (*url.URL, error)
		if conf.proxy != "" {
			proxyFunc = func(req *http.Request) (*url.URL, error) {
				config := &httpproxy.Config{
					HTTPProxy:  conf.proxy,
					HTTPSProxy: conf.proxy,
				}

				return config.ProxyFunc()(req.URL)
			}
		} else if !conf.noProxyFromEnv {
			proxyFunc = http.ProxyFromEnvironment
		}

		var nextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper
		if !conf.http2 {
			nextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s %q failed: %w", conf.backendName, addr, err)
				}
				return conn, nil
			},
			Dial: (&net.Dialer{
				KeepAlive: 60 * time.Second,
			}).Dial,
			DisableCompression:    true,
			DisableKeepAlives:     conf.disableConnectionReuse,
			MaxConnsPerHost:       conf.maxConnections,
			Proxy:                 proxyFunc,
			ResponseHeaderTimeout: conf.ttfbTimeout,
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
