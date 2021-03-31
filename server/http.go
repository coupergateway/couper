package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/logging"
)

type muxers map[string]*Mux

// HTTPServer represents a configured HTTP server.
type HTTPServer struct {
	accessLog  *logging.AccessLog
	commandCtx context.Context
	evalCtx    *eval.Context
	listener   net.Listener
	log        logrus.FieldLogger
	muxers     muxers
	port       string
	settings   *config.Settings
	shutdownCh chan struct{}
	srv        *http.Server
	timings    *runtime.HTTPTimings
	uidFn      func() string
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(
	cmdCtx context.Context, evalCtx *eval.Context, log logrus.FieldLogger, settings *config.Settings,
	timings *runtime.HTTPTimings, srvConf runtime.ServerConfiguration) ([]*HTTPServer, func()) {
	var list []*HTTPServer

	for port, hosts := range srvConf {
		list = append(list, New(cmdCtx, evalCtx, log, settings, timings, port, hosts))
	}

	handleShutdownFn := func() {
		<-cmdCtx.Done()
		time.Sleep(timings.ShutdownDelay + timings.ShutdownTimeout) // wait for max amount, TODO: feedback per server
	}

	return list, handleShutdownFn
}

// New creates a configured HTTP server.
func New(
	cmdCtx context.Context, evalCtx *eval.Context, log logrus.FieldLogger, settings *config.Settings,
	timings *runtime.HTTPTimings, p runtime.Port, hosts runtime.Hosts) *HTTPServer {
	var uidFn func() string
	if settings.RequestIDFormat == "uuid4" {
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

	muxersList := make(muxers)
	for host, muxOpts := range hosts {
		mux := NewMux(muxOpts)
		mux.MustAddRoute(http.MethodGet, settings.HealthPath, handler.NewHealthCheck(settings.HealthPath, shutdownCh))

		muxersList[host] = mux
	}

	httpSrv := &HTTPServer{
		evalCtx:    evalCtx,
		accessLog:  logging.NewAccessLog(&logConf, log),
		commandCtx: cmdCtx,
		log:        log,
		muxers:     muxersList,
		port:       p.String(),
		settings:   settings,
		shutdownCh: shutdownCh,
		timings:    timings,
		uidFn:      uidFn,
	}

	srv := &http.Server{
		Addr:              ":" + p.String(),
		Handler:           httpSrv,
		IdleTimeout:       timings.IdleTimeout,
		ReadHeaderTimeout: timings.ReadHeaderTimeout,
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
	s.log.Infof("couper is serving: %s", ln.Addr().String())

	go s.listenForCtx()

	go func() {
		if err := s.srv.Serve(ln); err != nil {
			s.log.Errorf("%s: %v", ln.Addr().String(), err.Error())
		}
	}()
}

// Close closes the listener
func (s *HTTPServer) Close() error {
	return s.listener.Close()
}

func (s *HTTPServer) listenForCtx() {
	<-s.commandCtx.Done()

	logFields := logrus.Fields{
		"delay":    s.timings.ShutdownDelay.String(),
		"deadline": s.timings.ShutdownTimeout.String(),
	}

	s.log.WithFields(logFields).Warn("shutting down")
	close(s.shutdownCh)

	time.Sleep(s.timings.ShutdownDelay)
	ctx, cancel := context.WithTimeout(context.Background(), s.timings.ShutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.WithFields(logFields).Error(err)
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	uid := s.uidFn()
	ctx := context.WithValue(req.Context(), request.UID, uid)
	ctx = context.WithValue(ctx, request.XFF, req.Header.Get("X-Forwarded-For"))
	*req = *req.WithContext(ctx)

	req.Host = s.getHost(req)

	host, _, err := runtime.GetHostPort(req.Host)
	if err != nil {
		errors.DefaultHTML.ServeError(errors.InvalidRequest).ServeHTTP(rw, req)
	}

	mux, ok := s.muxers[host]
	if !ok {
		mux, ok = s.muxers["*"]
		if !ok {
			errors.DefaultHTML.ServeError(errors.Configuration).ServeHTTP(rw, req)
		}
	}

	h := mux.FindHandler(req)
	w := NewRWWrapper(rw,
		transport.ReClientSupportsGZ.MatchString(
			req.Header.Get(transport.AcceptEncodingHeader),
		),
		s.settings.SecureCookies,
	)
	rw = w

	clientReq := req.Clone(ctx)

	if err := s.setGetBody(h, clientReq); err != nil {
		mux.opts.ServerOptions.ServerErrTpl.ServeError(err).ServeHTTP(rw, req)
		return
	}

	ctx = s.evalCtx.WithClientRequest(req)
	*clientReq = *clientReq.WithContext(ctx)

	s.accessLog.ServeHTTP(rw, clientReq, h, startTime)

	w.Close() // Closes the GZ writer.
}

func (s *HTTPServer) setGetBody(h http.Handler, req *http.Request) error {
	outer := h
	if inner, protected := outer.(ac.ProtectedHandler); protected {
		outer = inner.Child()
	}

	if limitHandler, ok := outer.(handler.EndpointLimit); ok {
		if err := eval.SetGetBody(req, limitHandler.RequestLimit()); err != nil {
			return err
		}
	}
	return nil
}

// getHost configures the host from the incoming request host based on
// the xfh setting and listener port to be prepared for the http multiplexer.
func (s *HTTPServer) getHost(req *http.Request) string {
	host := req.Host
	if s.settings.XForwardedHost {
		host = req.Header.Get("X-Forwarded-Host")
	}

	host = strings.ToLower(host)

	if !strings.Contains(host, ":") {
		return s.cleanHostAppendPort(host)
	}

	h, _, err := net.SplitHostPort(host)
	if err != nil {
		return s.cleanHostAppendPort(host)
	}

	return s.cleanHostAppendPort(h)
}

func (s *HTTPServer) cleanHostAppendPort(host string) string {
	return strings.TrimSuffix(host, ".") + ":" + s.port
}
