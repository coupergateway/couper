package runtime

import (
	"net/http"

	"github.com/coupergateway/couper/config/runtime/server"
)

type MuxOptions struct {
	EndpointRoutes map[string]http.Handler
	FileRoutes     map[string]http.Handler
	SPARoutes      map[string]http.Handler
	ServerOptions  *server.Options
}

func NewMuxOptions() *MuxOptions {
	return &MuxOptions{
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
	}
}
