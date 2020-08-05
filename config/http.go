package config

import "time"

// HTTP configures the ingress http server.
type HTTP struct {
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ListenPort        int
}

// DefaultHTTP sets some defaults for the http server.
var DefaultHTTP = HTTP{
	IdleTimeout:       time.Second * 60,
	ReadHeaderTimeout: time.Second * 10,
	ListenPort:        8083,
}
