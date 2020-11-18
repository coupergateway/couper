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

	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/test"
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
	port       string
	shutdownCh chan struct{}
	srv        *http.Server
	uidFn      func() string
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(cmdCtx context.Context, log logrus.FieldLogger, conf *runtime.HTTPConfig, srvConf *runtime.ServerConfiguration) ([]*HTTPServer, func()) {
	var list []*HTTPServer

	for port, srvMux := range srvConf.PortOptions {
		list = append(list, New(cmdCtx, log, conf, port, srvMux))
	}

	handleShutdownFn := func() {
		<-cmdCtx.Done()
		time.Sleep(conf.Timings.ShutdownDelay + conf.Timings.ShutdownTimeout) // wait for max amount, TODO: feedback per server
	}

	return list, handleShutdownFn
}

// New creates a configured HTTP server.
func New(cmdCtx context.Context, log logrus.FieldLogger, conf *runtime.HTTPConfig, p runtime.Port, muxOpts *runtime.MuxOptions) *HTTPServer {
	if conf == nil {
		log.Fatal("missing httpConfig")
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
		port:       p.String(),
		shutdownCh: shutdownCh,
		uidFn:      uidFn,
	}

	srv := &http.Server{
		Addr:              ":" + p.String(),
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
	select {
	case <-s.commandCtx.Done():
		logFields := logrus.Fields{
			"delay":    s.config.Timings.ShutdownDelay.String(),
			"deadline": s.config.Timings.ShutdownTimeout.String(),
		}

		s.log.WithFields(logFields).Warn("shutting down")
		close(s.shutdownCh)

		// testHook - skip shutdownDelay
		if _, ok := s.commandCtx.Value(test.Key).(bool); ok {
			_ = s.srv.Shutdown(context.TODO())
			return
		}

		time.Sleep(s.config.Timings.ShutdownDelay)
		ctx, cancel := context.WithTimeout(context.Background(), s.config.Timings.ShutdownTimeout)
		defer cancel()
		if err := s.srv.Shutdown(ctx); err != nil {
			s.log.WithFields(logFields).Error(err)
		}
	}
}

func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	uid := s.uidFn()
	ctx := context.WithValue(req.Context(), request.UID, uid)
	*req = *req.WithContext(ctx)

	req.Host = s.getHost(req)

	h := s.mux.FindHandler(req)
	rw = NewHeaderWriter(
		NewBodyZipper(rw,
			handler.ReClientSupportsGZ.MatchString(req.Header.Get(handler.AEHeader))),
	)
	s.accessLog.ServeHTTP(rw, req, h, startTime)
}

// getHost configures the host from the incoming request host based on
// the xfh setting and listener port to be prepared for the http multiplexer.
func (s *HTTPServer) getHost(req *http.Request) string {
	host := req.Host
	if s.config.UseXFH {
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
