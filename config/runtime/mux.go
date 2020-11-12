package runtime

import (
	"net/http"
)

type MuxOptions struct {
	Hosts          hosts
	EndpointRoutes map[string]http.Handler
	FileRoutes     map[string]http.Handler
	SPARoutes      map[string]http.Handler
}

func NewMuxOptions(hostsMap hosts) *MuxOptions {
	if hostsMap == nil {
		hostsMap = make(hosts)
	}

	return &MuxOptions{
		Hosts:          hostsMap,
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
	}
}
