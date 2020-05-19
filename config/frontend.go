package config

import "net/http"

type Frontend struct {
	Backend  http.Handler
	Endpoint Endpoint `hcl:"endpoint,block"`
	Name     string   `hcl:"name,label"`
	// Path     string   `hcl:"path,attr"`
}
