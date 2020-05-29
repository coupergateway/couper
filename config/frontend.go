package config

import "net/http"

type Frontend struct {
	Endpoint []*Endpoint `hcl:"endpoint,block"`
	Files    Files       `hcl:"files,block"`
	Name     string      `hcl:"name,label"`
	BasePath string      `hcl:"base_path,attr"`

	instance http.Handler
}

func (f *Frontend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if f.instance != nil {
		f.instance.ServeHTTP(rw, req)
	}
}

func (f *Frontend) String() string {
	if f.instance != nil {
		return "File"
	}
	return ""
}
