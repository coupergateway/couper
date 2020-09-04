package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/logging"
)

// HTTPServer represents a configured HTTP server.
type HTTPServer struct {
	accessLog  *logging.AccessLog
	config     *runtime.HTTPConfig
	commandCtx context.Context
	log        logrus.FieldLogger
	listener   net.Listener
	muxes      runtime.HostHandlers
	srv        *http.Server
	uidFn      func() string
}

// NewServerList creates a list of all configured HTTP server.
func NewServerList(cmdCtx context.Context, log *logrus.Entry, conf *runtime.HTTPConfig, handlers runtime.EntrypointHandlers) []*HTTPServer {
	var list []*HTTPServer

	for port, hosts := range handlers {
		list = append(list, New(cmdCtx, log, conf, port, hosts))
	}

	return list
}

// New creates a configured HTTP server.
func New(cmdCtx context.Context, log *logrus.Entry, conf *runtime.HTTPConfig, p runtime.Port, hosts runtime.HostHandlers) *HTTPServer {
	// TODO: uuid package switch with global option
	uidFn := func() string {
		return xid.New().String()
	}

	// TODO: hcl conf
	logConf := *logging.DefaultConfig
	logConf.TypeFieldKey = "couper_access"

	httpSrv := &HTTPServer{
		accessLog:  logging.NewAccessLog(&logConf, log.Logger),
		config:     conf,
		commandCtx: cmdCtx,
		log:        log,
		muxes:      hosts,
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
	s.log.WithField("addr", ln.Addr().String()).Info("couper gateway is serving") // TODO: server name

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
	ctx := context.WithValue(req.Context(), request.UID, uid)
	*req = *req.WithContext(ctx)

	h := s.getHandler(req)
	if h == nil {
		h = errors.DefaultHTML.ServeError(errors.Configuration)
	}

	s.accessLog.ServeHTTP(NewHeaderWriter(rw), req, h)
}

func (s *HTTPServer) getHandler(req *http.Request) http.Handler {
	host := s.getHost(req)

	if _, ok := s.muxes[host]; !ok {
		if _, ok := s.muxes["*"]; !ok {
			*req = *req.Clone(context.WithValue(req.Context(), request.ServerName, "-"))
			return nil
		}
		host = "*"
	}

	*req = *req.Clone(context.WithValue(req.Context(), request.ServerName, s.muxes[host].Server.Name))

	return NewMuxer(s.muxes[host].Mux).Match(req)
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
