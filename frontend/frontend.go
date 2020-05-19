package frontend

import "net/http"

type Frontend struct {
	Backend  http.Handler
	Endpoint Endpoint
	Name     string
	Path     string
}
