package config

import "net/http"

var _ http.Handler = &Endpoint{}

type Endpoint struct {
	Path string `hcl:"path,label"`
	Type string `hcl:"type,label"`
	// Desc    string `hcl:"description"`
	Backend http.Handler
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	e.Backend.ServeHTTP(rw, req)
}
