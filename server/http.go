package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/rs/xid"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/logging"
)

// HTTPServer represents a configured HTTP server.
type HTTPServer struct {
	accessLog  *logging.AccessLog
	commandCtx context.Context
	config     *runtime.HTTPConfig
	listener   net.Listener
	log        logrus.FieldLogger
	mux        *Mux
	name       string
	shutdownCh chan struct{}
	srv        *http.Server
	uidFn      func() string
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(cmdCtx context.Context, log logrus.FieldLogger, conf *runtime.HTTPConfig, server runtime.Server) ([]*HTTPServer, func()) {
	var list []*HTTPServer

	for port, srvMux := range server {
		list = append(list, New(cmdCtx, log, conf, srvMux.Server.Name, port, srvMux.Mux))
	}

	handleShutdownFn := func() {
		<-cmdCtx.Done()
		time.Sleep(conf.Timings.ShutdownDelay + conf.Timings.ShutdownTimeout) // wait for max amount, TODO: feedback per server
	}

	return list, handleShutdownFn
}

// New creates a configured HTTP server.
func New(cmdCtx context.Context, log logrus.FieldLogger, conf *runtime.HTTPConfig, name string, p runtime.Port, muxOpts *runtime.MuxOptions) *HTTPServer {
	if conf == nil {
		panic("missing httpConfig")
	}

	var uidFn func() string
	if conf.RequestIDFormat == "uuid4" {
		uidFn = func() string {
			return uuid.NewV4().String()
		}
	} else {
		uidFn = func() string {
			return xid.New().String()
		}
	}

	logConf := *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_access"
	env.DecodeWithPrefix(&logConf, "ACCESS_")

	shutdownCh := make(chan struct{})

	mux := NewMux(muxOpts)
	mux.MustAddRoute(http.MethodGet, conf.HealthPath, handler.NewHealthCheck(conf.HealthPath, shutdownCh))

	httpSrv := &HTTPServer{
		accessLog:  logging.NewAccessLog(&logConf, log),
		commandCtx: cmdCtx,
		config:     conf,
		log:        log,
		mux:        mux,
		name:       name,
		shutdownCh: shutdownCh,
		uidFn:      uidFn,
	}

	srv := &http.Server{
		Addr:              ":" + string(p),
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
	s.log.Infof("couper is serving: %s %s", s.name, ln.Addr().String())

	go s.listenForCtx()

	go func() {
		if err := s.srv.Serve(ln); err != nil {
			s.log.Errorf("%s %s: %v", s.name, ln.Addr().String(), err.Error())
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
		logFields := logrus.Fields{
			"delay":    s.config.Timings.ShutdownDelay.String(),
			"deadline": s.config.Timings.ShutdownTimeout.String(),
		}
		if s.name != "" {
			logFields["server"] = s.name
		}
		s.log.WithFields(logFields).Warn("shutting down")
		close(s.shutdownCh)
		time.Sleep(s.config.Timings.ShutdownDelay)
		ctx, cancel := context.WithTimeout(context.Background(), s.config.Timings.ShutdownTimeout)
		defer cancel()
		if err := s.srv.Shutdown(ctx); err != nil {
			s.log.WithFields(logFields).Error(err)
		}
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	uid := s.uidFn()
	ctx := context.WithValue(req.Context(), request.UID, uid)
	ctx = context.WithValue(ctx, request.ServerName, s.name)
	*req = *req.WithContext(ctx)

	if s.config.UseXFH {
		req.Host = req.Header.Get("X-Forwarded-Host")
	}
	h := s.mux.FindHandler(req)
	s.accessLog.ServeHTTP(NewHeaderWriter(rw), req, h)
}
