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
	file       string
	srvOptions *server.Options
}

func NewSpa(bootstrapFile string, srvOpts *server.Options) (*Spa, error) {
	absPath, err := filepath.Abs(bootstrapFile)
	if err != nil {
		return nil, err
	}
	return &Spa{
		file:       absPath,
		srvOptions: srvOpts,
	}, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, err := os.Open(s.file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			s.srvOptions.ServerErrTpl.ServeError(errors.RouteNotFound).ServeHTTP(rw, req)
			return
		}

		s.srvOptions.ServerErrTpl.ServeError(errors.Configuration).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		s.srvOptions.ServerErrTpl.ServeError(errors.Configuration).ServeHTTP(rw, req)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) Options() *server.Options {
	return s.srvOptions
}

func (s *Spa) String() string {
	return "spa"
}
