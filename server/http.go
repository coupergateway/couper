package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/config"
)

const RequestIDKey = "requestID"

type HTTPServer struct {
	ctx      context.Context
	Frontend []interface{}
	log      *logrus.Entry
	srv      *http.Server
}

func New(ctx context.Context) *HTTPServer {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
		logrus.FieldKeyTime: "timestamp",
		logrus.FieldKeyMsg:  "message",
	}}

	httpSrv := &HTTPServer{ctx: ctx, log: logger.WithField("type", "couper")}

	srv := &http.Server{
		Addr: ":" + config.DefaultHTTP.ListenPort,
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(context.Background(), RequestIDKey, xid.New().String())
		},
		Handler:           httpSrv,
		IdleTimeout:       config.DefaultHTTP.IdleTimeout,
		ReadHeaderTimeout: config.DefaultHTTP.ReadHeaderTimeout,
	}

	httpSrv.srv = srv

	return httpSrv
}

func (s *HTTPServer) Listen() int {
	s.log.WithField("addr", s.srv.Addr).Info("couper gateway is serving")
	go s.listenForCtx()
	err := s.srv.ListenAndServe()
	if err != nil {
		s.log.Error(err)
		return 1
	}
	return 0
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
	rw.Header().Add("server", "couper.io")
	rw.Header().Add("X-Request-Id", uid)
	s.log.WithField("uid", uid).WithField("agent", req.Header.Get("User-Agent")).WithField("url", req.URL.String()).Info()
}
