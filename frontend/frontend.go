package frontend

import "net/http"

type Frontend struct {
	Backend http.Handler
	Name    string
	Path    string
}
