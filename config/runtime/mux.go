package runtime

import (
	"net/http"

	"github.com/avenga/couper/errors"
)

type MuxOptions struct {
	EndpointRoutes map[string]http.Handler
	FileRoutes     map[string]http.Handler
	SPARoutes      map[string]http.Handler
	ErrorTpl       *errors.Template
	Hosts          hosts
}

func NewMuxOptions(errorTpl *errors.Template, hostsMap hosts) *MuxOptions {
	if hostsMap == nil {
		hostsMap = make(hosts)
	}

	return &MuxOptions{
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
		ErrorTpl:       errorTpl,
		Hosts:          hostsMap,
	}
}
