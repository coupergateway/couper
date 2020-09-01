package logging

import (
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
)

var requestSerial uint64

type AccessLog struct {
	conf   *Config
	logger logrus.FieldLogger
}

func NewAccessLog(logger logrus.FieldLogger) *AccessLog {
	return &AccessLog{
		logger: logger,
	}
}

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler) {
	now := time.Now()
	statusReader := NewStatusReader(rw)
	rw = statusReader // TODO: rename to recorder?

	reqID := req.Context().Value(request.RequestID)

	fields := Fields{
		"timestamp": now.UTC(),
		"request": Fields{
			"headers": filterHeader(DefaultConfig.RequestHeaders, req.Header),
		},
		"serial":     nextSerial(),
		"request_id": reqID,
		"method":     req.Method,
		"proto":      req.Proto,
	}

	path := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	fields["request_path"] = path.String()

	if req.Host != "" {
		fields["request_addr"] = req.Host
		fields["request_host"], fields["request_port"] = splitHostPort(req.Host)
	}

	fields["client_addr"] = req.RemoteAddr
	fields["client_host"], fields["client_port"] = splitHostPort(req.RemoteAddr)
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" { // TODO: if conf use xff
		fields["client_host"] = xff
	}

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	fields["scheme"] = scheme

	nextHandler.ServeHTTP(rw, req)

	fields["status"] = statusReader.status
	fields["response"] = Fields{
		"headers": filterHeader(DefaultConfig.ResponseHeaders, rw.Header()),
	}

	entry := log.logger.WithField("access", fields)
	if statusReader.status == http.StatusInternalServerError {
		entry.Error()
	} else {
		entry.Info()
	}
}

func filterHeader(list []string, src http.Header) http.Header {
	header := make(http.Header)
	for _, key := range list {
		ck := http.CanonicalHeaderKey(key)
		val, ok := src[http.CanonicalHeaderKey(ck)]
		if !ok || len(val) == 0 || val[0] == "" {
			continue
		}
		header[ck] = val
	}
	return header
}

func splitHostPort(hp string) (string, string) {
	host, port, err := net.SplitHostPort(hp)
	if err != nil {
		return hp, "-"
	}
	if port == "" {
		port = "-"
	}
	return host, port
}

func nextSerial() uint64 {
	return atomic.AddUint64(&requestSerial, 1)
}
