package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
)

const RequestIDKey = "requestID"

type HTTPServer struct {
	config *config.Gateway
	ctx    context.Context
	log    *logrus.Entry
	mux    *http.ServeMux
	srv    *http.Server
}

func New(ctx context.Context, conf *config.Gateway) *HTTPServer {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
		logrus.FieldKeyTime: "timestamp",
		logrus.FieldKeyMsg:  "message",
	}}

	httpSrv := &HTTPServer{ctx: ctx, config: conf, log: logger.WithField("type", "couper"), mux: http.NewServeMux()}

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
	for _, frontend := range s.config.Frontends {
		s.log.WithField("path", frontend.Endpoint.Path).Debug("registered")
		s.mux.Handle(frontend.Endpoint.Path, frontend.Endpoint.Backend)
	}
}

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
	handler, pattern := s.mux.Handler(req)
	rw.Header().Add("server", "couper.io")
	rw.Header().Add("X-Request-Id", uid)
	handler.ServeHTTP(rw, req)
	var handlerName string
	if name, ok := handler.(interface{ String() string }); ok {
		handlerName = name.String()
	}
	s.log.WithFields(logrus.Fields{
		"agent":   req.Header.Get("User-Agent"),
		"pattern": pattern,
		"handler": handlerName, // expected String() implementation
		"uid":     uid,
		"url":     req.URL.String(),
	}).Info()
}
