package config

import "net/http"

const (
	Proxy     = "Proxy"
	ServeDir  = "ServeDir"
	ServeFile = "ServeFile"
)

type Backend struct {
	Type string `hcl:"type,label"`
	// For easyness ... TODO: adapt hcl parsing to apply settings to instances
	OriginAddress string `hcl:"origin_address,optional"` // optional defaults to attr
	OriginHost    string `hcl:"origin_host,optional"`
	OriginScheme  string `hcl:"origin_scheme,optional"`

	instance http.Handler
}

func (b *Backend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.instance.ServeHTTP(rw, req)
}
