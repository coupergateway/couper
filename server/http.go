package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"

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

// NewServers returns a list of the created and configured HTTP(s) servers.
func NewServers(cmdCtx, evalCtx context.Context, log logrus.FieldLogger, settings *config.Settings,
	timings *runtime.HTTPTimings, srvConf runtime.ServerConfiguration) ([]*HTTPServer, func(), error) {

	var list []*HTTPServer

	for port, hosts := range srvConf {
		srv, err := New(cmdCtx, evalCtx, log, settings, timings, port, hosts)
		if err != nil {
			return nil, nil, err
		}
		list = append(list, srv)
	}

	handleShutdownFn := func() {
		<-cmdCtx.Done()
		time.Sleep(timings.ShutdownDelay + timings.ShutdownTimeout) // wait for max amount, TODO: feedback per server
	}

	return list, handleShutdownFn, nil
}

// New creates an HTTP(S) server with configured router and middlewares.
func New(cmdCtx, evalCtx context.Context, log logrus.FieldLogger, settings *config.Settings,
	timings *runtime.HTTPTimings, p runtime.Port, hosts runtime.Hosts) (*HTTPServer, error) {

	logConf := *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_access"
	env.DecodeWithPrefix(&logConf, "ACCESS_")

	shutdownCh := make(chan struct{})

	muxersList := make(muxers)
	var serverTLS *config.ServerTLS
	for host, muxOpts := range hosts {
		mux := NewMux(muxOpts)
		registerHandler(mux.endpointRoot, []string{http.MethodGet}, settings.HealthPath, handler.NewHealthCheck(settings.HealthPath, shutdownCh))
		mux.RegisterConfigured()
		muxersList[host] = mux

		// TODO: refactor (hosts,muxOpts, etc) format type and usage
		// serverOpts are all the same, pick first
		if serverTLS == nil && muxOpts.ServerOptions != nil && muxOpts.ServerOptions.TLS != nil {
			serverTLS = muxOpts.ServerOptions.TLS
		}
	}

	httpSrv := &HTTPServer{
		evalCtx:    evalCtx.Value(request.ContextType).(*eval.Context),
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
	telemetryHandler := middleware.NewHandler(httpSrv, nil) // fallback to plain wrapper without telemetry options
	if settings.TelemetryMetrics {
		telemetryHandler = middleware.NewMetricsHandler()(httpSrv)
	}
	if settings.TelemetryTraces {
		telemetryHandler = middleware.NewTraceHandler()(telemetryHandler)
	}

	uidHandler := middleware.NewUIDHandler(settings, httpsDevProxyIDField)(telemetryHandler)
	logHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		uidHandler.ServeHTTP(rw, req)
		accessLog.Do(rw, req)
	})
	recordHandler := middleware.NewRecordHandler(settings.SecureCookies)(logHandler)
	startTimeHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		recordHandler.ServeHTTP(rw, r.WithContext(
			context.WithValue(r.Context(), request.StartTime, time.Now())))
	})

	srv := &http.Server{
		Addr:              ":" + p.String(),
		ErrorLog:          newErrorLogWrapper(log),
		Handler:           startTimeHandler,
		IdleTimeout:       timings.IdleTimeout,
		ReadHeaderTimeout: timings.ReadHeaderTimeout,
	}

	if settings.TelemetryMetrics {
		srv.ConnState = httpSrv.onConnState
	}

	if serverTLS != nil {
		tlsConfig, err := newTLSConfig(serverTLS, log)
		if err != nil {
			return nil, err
		}
		srv.TLSConfig = tlsConfig
	}

	httpSrv.srv = srv

	return httpSrv, nil
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
		if s.srv.TLSConfig != nil {
			s.srv.Addr += "s"
		}
	}

	ln, err := net.Listen("tcp4", s.srv.Addr)
	if err != nil {
		return err
	}

	s.listener = ln
	s.log.Infof("couper is serving: %s", ln.Addr().String())

	go s.listenForCtx()

	go func() {
		var serveErr error
		if s.srv.TLSConfig != nil {
			serveErr = s.srv.ServeTLS(s.listener, "", "")
		} else {
			serveErr = s.srv.Serve(ln)
		}

		if serveErr != nil {
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
		h = errors.DefaultHTML.WithError(errors.ClientRequest)
	}

	mux, ok := s.muxers[host]
	if !ok {
		mux, ok = s.muxers["*"]
		if !ok && h == nil {
			h = errors.DefaultHTML.WithError(errors.Configuration)
		}
	}

	if h == nil {
		// mux.FindHandler() exchanges the req: *req = *req.WithContext(ctx)
		h = mux.FindHandler(req)
	}

	ctx := context.WithValue(req.Context(), request.LogEntry, s.log)
	ctx = context.WithValue(ctx, request.XFF, req.Header.Get("X-Forwarded-For"))

	// set innermost handler name for logging purposes
	if hs, stringer := getChildHandler(h).(fmt.Stringer); stringer {
		ctx = context.WithValue(ctx, request.Handler, hs.String())
	}

	if err = s.setGetBody(h, req); err != nil {
		h = mux.opts.ServerOptions.ServerErrTpl.WithError(err)
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
			portToAppend := req.URL.Port()
			req.URL.Host = xfh
			if portToAppend != "" && req.URL.Port() == "" {
				req.URL.Host += ":" + portToAppend
			}
		}
	}
	if s.settings.AcceptsForwardedPort() {
		if xfpo := req.Header.Get("X-Forwarded-Port"); xfpo != "" {
			req.URL.Host = req.URL.Hostname() + ":" + xfpo
		}
	}

	// due to the middleware callee stack we have to update the 'req' value.
	*req = *req.WithContext(s.evalCtx.WithClientRequest(req.WithContext(ctx)))

	h.ServeHTTP(rw, req)
}

func (s *HTTPServer) setGetBody(h http.Handler, req *http.Request) error {
	inner := getChildHandler(h)

	var err error
	if limitHandler, ok := inner.(handler.BodyLimit); ok {
		err = eval.SetGetBody(req, limitHandler.BufferOptions(), limitHandler.RequestLimit())
	}
	return err
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
	counter, _ := meter.SyncInt64().
		Counter(instrumentation.ClientConnectionsTotal, instrument.WithDescription(string(unit.Dimensionless)))
	gauge, _ := meter.SyncFloat64().UpDownCounter(
		instrumentation.ClientConnections,
		instrument.WithDescription(string(unit.Dimensionless)),
	)

	if state == http.StateNew {
		counter.Add(context.Background(), 1)
		gauge.Add(context.Background(), 1)
		// we have no callback for closing a hijacked one, so count them down too.
		// TODO: if required we COULD override given conn ptr value with own obj.
	} else if state == http.StateClosed || state == http.StateHijacked {
		gauge.Add(context.Background(), -1)
	}
}

// getChildHandler returns the innermost handler which supports the Child interface.
func getChildHandler(handler http.Handler) http.Handler {
	outer := handler
	for {
		if inner, ok := outer.(interface{ Child() http.Handler }); ok {
			outer = inner.Child()
			continue
		}
		break
	}
	return outer
}

// ErrorWrapper logs incoming Write bytes with the context filled logrus.FieldLogger.
type ErrorWrapper struct{ l logrus.FieldLogger }

func (e *ErrorWrapper) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.HasSuffix(msg, " tls: unknown certificate") {
		return len(p), nil // triggered on first browser connect for self signed certs; skip
	}

	e.l.Error(strings.TrimSpace(msg))
	return len(p), nil
}
func newErrorLogWrapper(logger logrus.FieldLogger) *log.Logger {
	return log.New(&ErrorWrapper{logger}, "", log.Lmsgprefix)
}
