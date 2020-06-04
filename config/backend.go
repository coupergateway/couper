package config

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

const (
	ServeDir  = "ServeDir"
	ServeFile = "ServeFile"
)

type Backend struct {
	Kind        string   `hcl:"kind,label"`
	Name        string   `hcl:"name,label"`
	Description string   `hcl:"description,optional"`
	Options     hcl.Body `hcl:",remain"`
	instance    http.Handler
}

func (b *Backend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.instance.ServeHTTP(rw, req)
}
