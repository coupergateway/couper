package config

import "net/http"

type Server struct {
	Name        string     `hcl:"name,label"`
	Backend     []*Backend `hcl:"backend,block"`
	BasePath    string     `hcl:"base_path,attr"`
	Domains     []string   `hcl:"domains,optional"`
	Files       Files      `hcl:"files,block"`
	Path        []*Path    `hcl:"path,block"`
	PathHandler PathHandler

	instance http.Handler
}

type PathHandler map[*Path]http.Handler

func (f *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if f.instance != nil {
		f.instance.ServeHTTP(rw, req)
	}
}

func (f *Server) String() string {
	if f.instance != nil {
		return "File"
	}
	return ""
}
