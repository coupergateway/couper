package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"

	"go.avenga.cloud/couper/gateway/config"
)

const RequestIDKey = "requestID"

type HTTPServer struct {
	config *config.Gateway
	ctx    context.Context
	log    *logrus.Entry
	mux    *mux.Router
	srv    *http.Server
}

func New(ctx context.Context, logger *logrus.Entry, conf *config.Gateway) *HTTPServer {
	httpSrv := &HTTPServer{ctx: ctx, config: conf, log: logger, mux: mux.NewRouter()}

	srv := &http.Server{
		Addr: ":" + config.DefaultHTTP.ListenPort,
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(context.Background(), RequestIDKey, xid.New().String())
		},
		Handler:           httpSrv,
		IdleTimeout:       config.DefaultHTTP.IdleTimeout,
		ReadHeaderTimeout: config.DefaultHTTP.ReadHeaderTimeout,
	}

	httpSrv.srv = srv

	return httpSrv
}

// registerHandler reads the given config frontends and register endpoints
// to our http multiplexer.
func (s *HTTPServer) registerHandler() {
	for _, server := range s.config.Server {
		for _, path := range server.Path {
			// Ensure we do not override the redirect behaviour due to the clean call from path.Join below.
			pattern := joinPath(server.BasePath, path.Pattern)
			s.log.WithField("server", server.Name).WithField("pattern", pattern).Debug("registered")
			s.mux.Handle(pattern, server.PathHandler[path])
		}
	}
}

// joinPath ensures the muxer behaviour for redirecting '/path' to '/path/' if not explicitly specified.
func joinPath(elements ...string) string {
	suffix := "/"
	if !strings.HasSuffix(elements[len(elements)-1], "/") {
		suffix = ""
	}
	return path.Join(elements...) + suffix
}

// Listen initiates the configured http handler and start listing on given port.
func (s *HTTPServer) Listen() int {
	s.log.WithField("addr", s.srv.Addr).Info("couper gateway is serving")
	s.registerHandler()
	go s.listenForCtx()
	err := s.srv.ListenAndServe()
	if err != nil {
		s.log.Error(err)
		return 1
	}
	return 0
}

func (s *HTTPServer) listenForCtx() {
	select {
	case <-s.ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		s.log.WithField("deadline", "10s").Warn("shutting down")
		s.srv.Shutdown(ctx)
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	uid := req.Context().Value(RequestIDKey).(string)
	req.Header.Set("X-Request-Id", uid)
	handler := s.mux
	rw.Header().Set("server", "couper.io") // TODO: wrap 'rw' for server override and status readout
	rw.Header().Set("X-Request-Id", uid)

	if fmt.Sprintf("%p", handler) == fmt.Sprintf("%p", http.NotFound) {
		//FIXME ??? handler = s.config.Server[0]
	}

	sr := &StatusReader{rw: rw}
	handler.ServeHTTP(sr, req)

// 	var handlerName string
// 	if name, ok := handler.(interface{ String() string }); ok {
// 		handlerName = name.String()
// 	}
	s.log.WithFields(logrus.Fields{
		"agent":   req.Header.Get("User-Agent"),
// 		"pattern": pattern,
// 		"handler": handlerName,
		"status":  sr.status,
		"uid":     uid,
		"url":     req.URL.String(),
	}).Info()
}
