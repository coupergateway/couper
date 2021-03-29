package runtime

import (
	"net/http"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

type MuxOptions struct {
	EndpointRoutes map[string]http.Handler
	FileRoutes     map[string]http.Handler
	SPARoutes      map[string]http.Handler
	APIErrorTpls   map[*config.API]*errors.Template
	ServerErrorTpl *errors.Template
	FilesErrorTpl  *errors.Template
	APIBasePaths   map[*config.API]string
	FilesBasePath  string
	SPABasePath    string
	ServerName     string
}

func NewMuxOptions() *MuxOptions {
	return &MuxOptions{
		EndpointRoutes: make(map[string]http.Handler),
		FileRoutes:     make(map[string]http.Handler),
		SPARoutes:      make(map[string]http.Handler),
	}
}
