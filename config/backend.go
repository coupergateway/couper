package config

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"
)

const (
	Proxy     = "Proxy"
	ServeDir  = "ServeDir"
	ServeFile = "ServeFile"
)

type Backend struct {
	Type        string   `hcl:",label"`
	Description string   `hcl:"description,optional"`
	Options     hcl.Body `hcl:",remain"`
	instance    http.Handler
}

func (b *Backend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.instance.ServeHTTP(rw, req)
}
