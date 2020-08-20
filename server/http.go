package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/runtime"
	"go.avenga.cloud/couper/gateway/errors"
)

type HTTPServer struct {
	config   *config.Gateway
	ctx      context.Context
	log      *logrus.Entry
	listener net.Listener
	srv      *http.Server
	uidFn    func() string
}

func New(ctx context.Context, logger *logrus.Entry, conf *config.Gateway) *HTTPServer {
	configure(conf, logger)
	// TODO: uuid package switch with global option
	uidFn := func() string {
		return xid.New().String()
	}

	httpSrv := &HTTPServer{
		ctx:    ctx,
		config: conf,
		log:    logger,
		uidFn:  uidFn,
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", conf.ListenPort),
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
	uid := s.uidFn()
	ctx := context.WithValue(req.Context(), runtime.RequestID, uid)
	*req = *req.WithContext(ctx)

	req.Header.Set("X-Request-Id", uid)
	rw.Header().Set("X-Request-Id", uid)

	h := s.getHandler(req)

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
		errors.DefaultHTML.ServeError(errors.ConfigurationError).ServeHTTP(sr, req)
		err = fmt.Errorf("%w: %s", errors.ConfigurationError, req.URL.String())
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

func (s *HTTPServer) getHandler(req *http.Request) http.Handler {
	host, port := s.getHostPort(req)

	if _, ok := s.config.Lookups[port]; !ok {
		return nil
	}
	if _, ok := s.config.Lookups[port][host]; !ok {
		if _, ok := s.config.Lookups[port]["*"]; !ok {
			return nil
		}

		host = "*"
	}

	return NewMuxer(s.config.Lookups[port][host].Mux).Match(req)
}

func (s *HTTPServer) getHostPort(req *http.Request) (string, string) {
	host := req.Host
	if strings.IndexByte(host, ':') == -1 {
		return host, fmt.Sprintf("%d", s.config.ListenPort)
	}

	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return host, fmt.Sprintf("%d", s.config.ListenPort)
	}

	return h, p
}
