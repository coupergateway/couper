package config

import "net/http"

type Frontend interface {
	http.Handler
	Endpoint() http.Handler
	Name() string
}

type Gateway struct {
	Frontends []Frontend
}
