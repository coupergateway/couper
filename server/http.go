package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/unit"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/telemetry/instrumentation"
	"github.com/avenga/couper/telemetry/provider"
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
		evalCtx:    evalCtx.Value(request.ContextType).(*eval.Context),
		accessLog:  logging.NewAccessLog(&logConf, log),
		commandCtx: cmdCtx,
		log:        log,
		muxers:     muxersList,
		port:       p.String(),
		settings:   settings,
		shutdownCh: shutdownCh,
		timings:    timings,
	}

	accessLog := logging.NewAccessLog(&logConf, log)

	// order matters
	traceHandler := middleware.NewTraceHandler()(httpSrv)
	uidHandler := middleware.NewUIDHandler(settings, httpsDevProxyIDField)(traceHandler)
	logHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		accessLog.ServeHTTP(rw, req, uidHandler)
	})
	recordHandler := middleware.NewRecordHandler(settings.SecureCookies)(logHandler)
	startTimeHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		recordHandler.ServeHTTP(rw, r.WithContext(
			context.WithValue(r.Context(), request.StartTime, time.Now())))
	})

	srv := &http.Server{
		Addr:              ":" + p.String(),
		ConnState:         httpSrv.onConnState,
		Handler:           startTimeHandler,
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
	var h http.Handler

	req.Host = s.getHost(req)
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

	if err = s.setGetBody(h, req); err != nil {
		h = mux.opts.ServerOptions.ServerErrTpl.ServeError(err)
	}

	req.URL.Host = req.Host
	req.URL.Scheme = "http"
	if s.settings.AcceptsForwardedProtocol() {
		if xfpr := req.Header.Get("X-Forwarded-Proto"); xfpr != "" {
			req.URL.Scheme = xfpr
			req.URL.Host = req.URL.Hostname()
		}
	}
	if s.settings.AcceptsForwardedHost() {
		if xfh := req.Header.Get("X-Forwarded-Host"); xfh != "" {
			req.URL.Host = xfh
			if req.URL.Port() != "" {
				req.URL.Host += ":" + req.URL.Port()
			}
		}
	}
	if s.settings.AcceptsForwardedPort() {
		if xfpo := req.Header.Get("X-Forwarded-Port"); xfpo != "" {
			req.URL.Host = req.URL.Hostname() + ":" + xfpo
		}
	}

	ctx := context.WithValue(req.Context(), request.XFF, req.Header.Get("X-Forwarded-For"))
	ctx = context.WithValue(ctx, request.LogEntry, s.log)
	if hs, stringer := h.(fmt.Stringer); stringer {
		ctx = context.WithValue(ctx, request.Handler, hs.String())
	}

	// due to the middleware callee stack we have to update the 'req' value.
	*req = *req.WithContext(s.evalCtx.WithClientRequest(req.WithContext(ctx)))

	h.ServeHTTP(rw, req)
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

func (s *HTTPServer) onConnState(_ net.Conn, state http.ConnState) {
	meter := provider.Meter("couper/server")
	counter := metric.Must(meter).NewInt64Counter(instrumentation.ClientConnectionsTotal, metric.WithDescription(string(unit.Dimensionless)))
	gauge := metric.Must(meter).NewFloat64UpDownCounter(instrumentation.ClientConnections, metric.WithDescription(string(unit.Dimensionless)))

	if state == http.StateNew {
		meter.RecordBatch(context.Background(), nil,
			counter.Measurement(1),
			gauge.Measurement(1),
		)
		// we have no callback for closing a hijacked one, so count them down too.
		// TODO: if required we COULD override given conn ptr value with own obj.
	} else if state == http.StateClosed || state == http.StateHijacked {
		meter.RecordBatch(context.Background(), nil,
			gauge.Measurement(-1),
		)
	}
}
