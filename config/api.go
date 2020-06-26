package config

import "net/http"

type Api struct {
	BasePath    string     `hcl:"base_path,optional"`
	Backend     []*Backend `hcl:"backend,block"`
	Path        []*Path    `hcl:"path,block"`
	PathHandler PathHandler
}

type PathHandler map[*Path]http.Handler
