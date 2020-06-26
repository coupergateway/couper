package config

import "net/http"

type Api struct {
	BasePath    string      `hcl:"base_path,optional"`
	Backend     []*Backend  `hcl:"backend,block"`
	Endpoint    []*Endpoint `hcl:"endpoint,block"`
	PathHandler PathHandler
}

type PathHandler map[*Endpoint]http.Handler
