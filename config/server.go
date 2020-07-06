package config

import "net/http"

type Server struct {
	Name        string     `hcl:"name,label"`
	Backend     []*Backend `hcl:"backend,block"`
	BasePath    string     `hcl:"base_path,optional"`
	Domains     []string   `hcl:"domains,optional"`
	Files       Files      `hcl:"files,block"`
	Path        []*Path    `hcl:"path,block"`
	PathHandler PathHandler
	FileHandler http.Handler
}

type PathHandler map[*Path]http.Handler
