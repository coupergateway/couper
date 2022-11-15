package test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// NewHTTPClient creates a new <http.Client> object.
func NewHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: time.Second * 5,
	}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, _ := net.SplitHostPort(addr)
				if port != "" {
					return dialer.DialContext(ctx, "tcp4", "127.0.0.1:"+port)
				}
				return dialer.DialContext(ctx, "tcp4", "127.0.0.1")
			},
			DisableCompression: true,
		},
	}
}

func NewHTTPSClient(tlsConf *tls.Config) *http.Client {
	client := NewHTTPClient()
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = tlsConf
	client.Transport = transport
	return client
}
