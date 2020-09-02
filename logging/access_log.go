package logging

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
)

type AccessLog struct {
	conf   *Config
	logger logrus.FieldLogger
}

func NewAccessLog(c *Config, logger logrus.FieldLogger) *AccessLog {
	return &AccessLog{
		conf:   c,
		logger: logger,
	}
}

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler) {
	now := time.Now()
	statusRecorder := NewStatusRecorder(rw)
	rw = statusRecorder

	uniqueID := req.Context().Value(request.RequestID)
	connectionSerial := req.Context().Value(request.ConnectionSerial)

	requestFields := Fields{
		"headers": filterHeader(log.conf.RequestHeaders, req.Header),
	}

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	fields := Fields{
		"connection_serial": connectionSerial,
		"method":            req.Method,
		"proto":             req.Proto,
		"request":           requestFields,
		"timestamp":         now.UTC(),
		"uid":               uniqueID,
	}

	if log.conf.TypeFieldKey != "" {
		fields["type"] = log.conf.TypeFieldKey
	}

	path := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	requestFields["path"] = path.String()

	if req.Host != "" {
		requestFields["addr"] = req.Host
		requestFields["host"], requestFields["port"] = splitHostPort(req.Host)
	}

	clientFields := Fields{
		"addr": req.RemoteAddr,
	}
	fields["client"] = clientFields
	clientFields["host"], clientFields["port"] = splitHostPort(req.RemoteAddr)
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" { // TODO: if conf use xff
		clientFields["host"] = xff
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

	fields["status"] = statusRecorder.status

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, rw.Header()),
	}
	fields["response"] = responseFields

	if statusRecorder.writtenBytes > 0 {
		responseFields["bytes"] = statusRecorder.writtenBytes
	}

	var entry *logrus.Entry
	if log.conf.ParentFieldKey != "" {
		entry = log.logger.WithField(log.conf.ParentFieldKey, fields)
	} else {
		entry = log.logger.WithFields(logrus.Fields(fields))
	}
	if statusRecorder.status == http.StatusInternalServerError {
		entry.Error()
	} else {
		entry.Info()
	}
}

func filterHeader(list []string, src http.Header) map[string]string {
	header := make(map[string]string)
	for _, key := range list {
		ck := http.CanonicalHeaderKey(key)
		val, ok := src[http.CanonicalHeaderKey(ck)]
		if !ok || len(val) == 0 || val[0] == "" {
			continue
		}
		header[strings.ToLower(key)] = strings.Join(val, "|")
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
