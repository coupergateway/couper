package transport

import (
	"context"
	"net"
	"net/http"
	"time"
)

type probeOptions struct {
	t time.Duration
}

func newClient(c *Config) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", c.Origin)
			},
			DisableCompression: true,
		},
	}
}

func (c *Config) Probe(b *Backend) {
	counter := 0
	probeOpts := &probeOptions{
		t: time.Second,
	}
	req, _ := http.NewRequest(http.MethodGet, "", nil)
	for {
		time.Sleep(probeOpts.t)
		c, _ = b.evalTransport(req)
		req, _ = http.NewRequest(http.MethodGet, c.Scheme+"://"+c.Origin, nil)
		_, err := newClient(c).Do(req)
		println("healthcheck:", counter, err == nil)
		counter++
	}
}
