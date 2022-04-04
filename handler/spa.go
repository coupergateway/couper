package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/server/writer"
)

var (
	_ http.Handler   = &Spa{}
	_ server.Context = &Spa{}
)

type Spa struct {
	file       string
	modifier   []hcl.Body
	srvOptions *server.Options
}

func NewSpa(bootstrapFile string, srvOpts *server.Options, modifier []hcl.Body) (*Spa, error) {
	absPath, err := filepath.Abs(bootstrapFile)
	if err != nil {
		return nil, err
	}
	return &Spa{
		file:       absPath,
		modifier:   modifier,
		srvOptions: srvOpts,
	}, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	file, err := os.Open(s.file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			s.srvOptions.ServerErrTpl.WithError(errors.RouteNotFound).ServeHTTP(rw, req)
			return
		}

		s.srvOptions.ServerErrTpl.WithError(errors.Configuration).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		s.srvOptions.ServerErrTpl.WithError(errors.Configuration).ServeHTTP(rw, req)
		return
	}

	if r, ok := rw.(*writer.Response); ok {
		evalContext := eval.ContextFromRequest(req)
		r.AddModifier(evalContext.HCLContext(), s.modifier...)
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) Options() *server.Options {
	return s.srvOptions
}

func (s *Spa) String() string {
	return "spa"
}
