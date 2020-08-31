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
	"go.avenga.cloud/couper/gateway/config/request"
	"go.avenga.cloud/couper/gateway/config/runtime"
	"go.avenga.cloud/couper/gateway/errors"
)

// HTTPServer represents a configured HTTP server.
type HTTPServer struct {
	config     *runtime.HTTPConfig
	commandCtx context.Context
	log        *logrus.Entry
	listener   net.Listener
	muxes      runtime.Hosts
	port       string
	srv        *http.Server
	uidFn      func() string
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(cmdCtx context.Context, logger *logrus.Entry, conf *runtime.HTTPConfig) []*HTTPServer {
	runtime.ConfigureHCL(conf, logger)

	var list []*HTTPServer

	for port, hosts := range conf.Lookups {
		list = append(list, New(cmdCtx, logger, conf, port, hosts))
	}

	return list
}

// New creates a configured HTTP server.
func New(cmdCtx context.Context, logger *logrus.Entry, conf *runtime.HTTPConfig, port string, hosts runtime.Hosts) *HTTPServer {

	// TODO: uuid package switch with global option
	uidFn := func() string {
		return xid.New().String()
	}
	httpSrv := &HTTPServer{
		config:     conf,
		commandCtx: cmdCtx,
		log:        logger,
		muxes:      hosts,
		port:       port,
		uidFn:      uidFn,
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpSrv,
		IdleTimeout:       conf.Timings.IdleTimeout,
		ReadHeaderTimeout: conf.Timings.ReadHeaderTimeout,
	}

	httpSrv.srv = srv

	return httpSrv
}

// Addr returns the listener address.
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
	case <-s.commandCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		s.log.WithField("deadline", "10s").Warn("shutting down")
		s.srv.Shutdown(ctx)
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	uid := s.uidFn()
	ctx := context.WithValue(req.Context(), request.RequestID, uid)
	*req = *req.WithContext(ctx)

	req.Header.Set("X-Request-Id", uid)
	rw.Header().Set("X-Request-Id", uid)

	srv, h := s.getHandler(req)

	var err error
	var handle, handlerName string
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

	if srv != nil {
		handle = srv.Name
	}

	fields := logrus.Fields{
		"agent":   req.Header.Get("User-Agent"),
		"handle":  handle,
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

func (s *HTTPServer) getHandler(req *http.Request) (*config.Server, http.Handler) {
	host := s.getHost(req)

	if _, ok := s.muxes[host]; !ok {
		if _, ok := s.muxes["*"]; !ok {
			return nil, nil
		}

		host = "*"
	}

	return s.muxes[host].Server, NewMuxer(s.muxes[host].Mux).Match(req)
}

func (s *HTTPServer) getHost(req *http.Request) string {
	host := req.Host
	if s.config.UseXFH {
		host = req.Header.Get("X-Forwarded-Host")
	}

	if strings.IndexByte(host, ':') == -1 {
		return cleanHost(host)
	}

	h, _, err := net.SplitHostPort(host)
	if err != nil {
		return cleanHost(host)
	}

	return cleanHost(h)
}

func cleanHost(host string) string {
	return strings.TrimRight(host, ".")
}
