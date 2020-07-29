package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/assets"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
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
	_, ph := configure(conf, logger)
	httpSrv := &HTTPServer{ctx: ctx, config: conf, log: logger, mux: NewMux(conf, ph)}

	addr := fmt.Sprintf(":%d", DefaultHTTPConfig.ListenPort)
	if conf.Addr != "" {
		addr = conf.Addr
	}
	srv := &http.Server{
		Addr: addr,
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(context.Background(), RequestIDKey, xid.New().String())
		},
		Handler:           httpSrv,
		IdleTimeout:       DefaultHTTPConfig.IdleTimeout,
		ReadHeaderTimeout: DefaultHTTPConfig.ReadHeaderTimeout,
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
		s.log.Fatal(err)
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

// Close closes the listener
func (s *HTTPServer) Close() error {
	return s.listener.Close()
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
	rw.Header().Set("X-Request-Id", uid)

	h := s.mux.Match(req)

	var err error
	var handlerName string
	sr := NewStatusReader(rw)
	if h != nil {
		h.ServeHTTP(sr, req)
		if name, ok := h.(interface{ String() string }); ok {
			handlerName = name.String()
		}
	} else {
		handlerName = "none"
		asset := assets.Assets.MustOpen("error.html")
		handler.NewErrorHandler(asset, 1001, http.StatusInternalServerError).ServeHTTP(rw, req)
		err = errors.New("no configuration found: " + req.URL.String())
	}

	fields := logrus.Fields{
		"agent":   req.Header.Get("User-Agent"),
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
