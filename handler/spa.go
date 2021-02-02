package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
)

var (
	_ http.Handler   = &Spa{}
	_ server.Context = &Spa{}
)

type Spa struct {
	cors       *CORSOptions
	file       string
	srvOptions *server.Options
}

func NewSpa(
	bootstrapFile string, srvOpts *server.Options, corsOpts *CORSOptions,
) (*Spa, error) {
	absPath, err := filepath.Abs(bootstrapFile)
	if err != nil {
		return nil, err
	}
	return &Spa{
		cors:       corsOpts,
		file:       absPath,
		srvOptions: srvOpts,
	}, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if isCorsPreflightRequest(req) {
		setCorsRespHeaders(s.cors, rw.Header(), req)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, err := os.Open(s.file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			s.srvOptions.ServerErrTpl.ServeError(errors.SPARouteNotFound).ServeHTTP(rw, req)
			return
		}

		s.srvOptions.ServerErrTpl.ServeError(errors.SPAError).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		s.srvOptions.ServerErrTpl.ServeError(errors.SPAError).ServeHTTP(rw, req)
		return
	}

	setCorsRespHeaders(s.cors, rw.Header(), req)

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) Options() *server.Options {
	return s.srvOptions
}

func (s *Spa) String() string {
	return "spa"
}
