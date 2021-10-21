package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
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
	bodies     []hcl.Body
	file       string
	logger     *logrus.Entry
	modifier   []hcl.Body
	srvOptions *server.Options
}

func NewSpa(bootstrapFile string, srvOpts *server.Options, modifier []hcl.Body, body hcl.Body, logger *logrus.Entry) (*Spa, error) {
	absPath, err := filepath.Abs(bootstrapFile)
	if err != nil {
		return nil, err
	}

	return &Spa{
		bodies:     append(srvOpts.Bodies, body),
		file:       absPath,
		logger:     logger,
		modifier:   modifier,
		srvOptions: srvOpts,
	}, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	evalContext := eval.ContextFromRequest(req)
	eval.ApplyCustomLogs(evalContext, s.bodies, req, s.logger, request.AccessLogFields)

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

	if r, ok := rw.(*writer.Response); ok {
		evalContext := eval.ContextFromRequest(req)
		r.AddModifier(evalContext, s.modifier)
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) Options() *server.Options {
	return s.srvOptions
}

func (s *Spa) String() string {
	return "spa"
}
