package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
)

const RequestIDKey = "requestID"

type HTTPServer struct {
	config   *config.Gateway
	ctx      context.Context
	log      *logrus.Entry
	listener net.Listener
	mux      *Mux
	srv      *http.Server
}

func New(ctx context.Context, logger *logrus.Entry, conf *config.Gateway) *HTTPServer {
	httpSrv := &HTTPServer{ctx: ctx, config: conf, log: logger, mux: NewMux(conf)}

	addr := ":" + config.DefaultHTTP.ListenPort
	if conf.Addr != "" {
		addr = conf.Addr
	}
	srv := &http.Server{
		Addr: addr,
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

func (s *HTTPServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// Listen initiates the configured http handler and start listing on given port.
func (s *HTTPServer) Listen() {
	if s.srv.Addr == "" {
		s.srv.Addr = ":http"
	}
	ln, err := net.Listen("tcp4", s.srv.Addr)
	if err != nil {
		s.log.Error(err)
		return
	}
	s.listener = ln
	s.log.WithField("addr", ln.Addr().String()).Info("couper gateway is serving")

	go s.listenForCtx()

	go func() {
		if err := s.srv.Serve(ln); err != nil {
			s.log.Error(err)
		}
	}()
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
	rw.Header().Set("server", "couper.io")
	rw.Header().Set("X-Request-Id", uid)

	handler, pattern := s.mux.Match(req)

	var err error
	var handlerName string
	sr := &StatusReader{rw: rw}
	if handler != nil {
		handler.ServeHTTP(sr, req)
		if name, ok := handler.(interface{ String() string }); ok {
			handlerName = name.String()
		}
	} else {
		handlerName = "none"
		sr.WriteHeader(http.StatusInternalServerError)
		err = errors.New("no configuration found: " + req.URL.String())
	}

	fields := logrus.Fields{
		"agent":   req.Header.Get("User-Agent"),
		"pattern": pattern,
		"handler": handlerName,
		"status":  sr.status,
		"uid":     uid,
		"url":     req.URL.String(),
	}

	if sr.status == http.StatusInternalServerError {
		s.log.WithFields(fields).Error(err)
	} else {
		s.log.WithFields(fields).Info()
	}
}
