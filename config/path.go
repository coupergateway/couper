package config

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

var _ http.Handler = &Path{}

type Path struct {
	Application *Application // parent
	Backend     http.Handler // `hcl:"backend,block"`
	Pattern     string       `hcl:"path,label"`
	Kind        string       `hcl:"kind,label"`
	Options     hcl.Body     `hcl:",remain"`
}

func (e *Path) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// TODO: override options
	e.Backend.ServeHTTP(rw, req)
}

func (e *Path) String() string {
	return e.Application.Name + ": " + e.Kind
}
