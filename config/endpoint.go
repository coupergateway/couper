package config

import "net/http"

var _ http.Handler = &Endpoint{}

type Endpoint struct {
	Backend  *Backend  `hcl:"backend,block"`
	Frontend *Frontend // parent
	Path     string    `hcl:"path,label"`
	// Type    string       `hcl:"type,label"` // 2nd arg in block statement
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	e.Backend.ServeHTTP(rw, req)
}

func (e *Endpoint) String() string {
	return e.Frontend.Name + ": " + e.Backend.Type
}
