package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
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
	uidFn      uidFunc
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(cmdCtx, evalCtx context.Context, log logrus.FieldLogger, settings *config.Settings,
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
func New(cmdCtx, evalCtx context.Context, log logrus.FieldLogger, settings *config.Settings,
	timings *runtime.HTTPTimings, p runtime.Port, hosts runtime.Hosts) *HTTPServer {

	uidFn := newUIDFunc(settings)

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
		evalCtx:    evalCtx.Value(eval.ContextType).(*eval.Context),
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
func (s *HTTPServer) Listen() error {
	if s.srv.Addr == "" {
		s.srv.Addr = ":http"
	}
	ln, err := net.Listen("tcp4", s.srv.Addr)
	if err != nil {
		return err
	}

	s.listener = ln
	s.log.Infof("couper is serving: %s", ln.Addr().String())

	go s.listenForCtx()

	go func() {
		if serveErr := s.srv.Serve(ln); serveErr != nil {
			if serveErr == http.ErrServerClosed {
				s.log.Infof("%v: %s", serveErr, ln.Addr().String())
			} else {
				s.log.Errorf("%s: %v", ln.Addr().String(), serveErr)
			}
		}
	}()
	return nil
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
	ctx := context.Background()
	if s.timings.ShutdownTimeout > 0 {
		c, cancel := context.WithTimeout(ctx, s.timings.ShutdownTimeout)
		defer cancel()
		ctx = c
	}

	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.WithFields(logFields).Error(err)
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	if err := s.setUID(rw, req); err != nil {
		s.accessLog.ServeHTTP(rw, req, errors.DefaultHTML.ServeError(err), startTime)
		return
	}

	ctx := context.WithValue(req.Context(), request.XFF, req.Header.Get("X-Forwarded-For"))
	*req = *req.WithContext(ctx)

	req.Host = s.getHost(req)

	gw := writer.NewGzipWriter(rw, req.Header)
	w := writer.NewResponseWriter(gw, s.settings.SecureCookies)
	// This defer closes the GZ writer but more important is triggering our own buffer logic in all cases
	// for this writer to prevent the 200 OK status fallback (http.ResponseWriter) and an empty response body.
	defer gw.Close()

	var h http.Handler

	host, _, err := runtime.GetHostPort(req.Host)
	if err != nil {
		h = errors.DefaultHTML.ServeError(errors.ClientRequest)
	}

	mux, ok := s.muxers[host]
	if !ok {
		mux, ok = s.muxers["*"]
		if !ok && h == nil {
			h = errors.DefaultHTML.ServeError(errors.Configuration)
		}
	}

	if h == nil {
		h = mux.FindHandler(req)
	}

	clientReq := req.Clone(req.Context())

	if err = s.setGetBody(h, clientReq); err != nil {
		h = mux.opts.ServerOptions.ServerErrTpl.ServeError(err)
	}

	clientReq.URL.Host = req.Host
	clientReq.URL.Scheme = "http"
	if s.settings.AcceptsForwardedProtocol() {
		if xfpr := req.Header.Get("X-Forwarded-Proto"); xfpr != "" {
			clientReq.URL.Scheme = xfpr
			clientReq.URL.Host = clientReq.URL.Hostname()
		}
	}
	if s.settings.AcceptsForwardedHost() {
		if xfh := req.Header.Get("X-Forwarded-Host"); xfh != "" {
			clientReq.URL.Host = xfh
			if clientReq.URL.Port() != "" {
				clientReq.URL.Host += ":" + clientReq.URL.Port()
			}
		}
	}
	if s.settings.AcceptsForwardedPort() {
		if xfpo := req.Header.Get("X-Forwarded-Port"); xfpo != "" {
			clientReq.URL.Host = clientReq.URL.Hostname() + ":" + xfpo
		}
	}

	ctx = s.evalCtx.WithClientRequest(clientReq)
	*clientReq = *clientReq.WithContext(ctx)

	s.accessLog.ServeHTTP(w, clientReq, h, startTime)
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
		if xfh := req.Header.Get("X-Forwarded-Host"); xfh != "" {
			host = xfh
		}
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
