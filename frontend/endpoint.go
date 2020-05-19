package frontend

import "net/http"

var _ http.Handler = &Endpoint{}

type Endpoint struct {
	Path    string
	Backend http.Handler
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	e.Backend.ServeHTTP(rw, req)
}
