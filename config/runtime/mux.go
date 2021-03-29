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
}

func NewMuxOptions(errorTpl *errors.Template) *MuxOptions {
	return &MuxOptions{
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
		ErrorTpl:       errorTpl,
	}
}
