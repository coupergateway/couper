package logging

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

type RoundtripHandlerFunc http.HandlerFunc

func (f RoundtripHandlerFunc) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f(rw, req)
}

type AccessLog struct {
	conf   *Config
	logger logrus.FieldLogger
}

type RecorderInfo interface {
	StatusCode() int
	WrittenBytes() int
}

func NewAccessLog(c *Config, logger logrus.FieldLogger) *AccessLog {
	conf := c
	if conf == nil {
		conf = DefaultConfig
	}
	return &AccessLog{
		conf:   conf,
		logger: logger,
	}
}

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler, startTime time.Time) {
	nextHandler.ServeHTTP(rw, req)
	serveDone := time.Now()

	fields := Fields{}

	backendName, _ := req.Context().Value(request.BackendName).(string)
	if backendName == "" {
		endpointName, _ := req.Context().Value(request.Endpoint).(string)
		if endpointName != "" {
			fields["endpoint"] = endpointName
		}
	}

	fields["method"] = req.Method
	fields["server"] = req.Context().Value(request.ServerName)
	fields["uid"] = req.Context().Value(request.UID)

	requestFields := Fields{
		"headers": filterHeader(log.conf.RequestHeaders, req.Header),
		"method":  req.Method,
		"proto":   "https",
	}
	fields["request"] = requestFields

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	// Read out handler kind from stringer interface
	if h, ok := nextHandler.(fmt.Stringer); ok && h.String() != "" {
		fields["handler"] = h.String()
	} else if kind, k := req.Context().Value(request.EndpointKind).(string); k { // fallback, e.g. with ErrorHandler
		fields["handler"] = kind
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

	requestFields["origin"] = req.URL.Host
	requestFields["host"], fields["port"] = splitHostPort(req.URL.Host)

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	var statusCode int
	var writtenBytes int
	if recorder, ok := rw.(RecorderInfo); ok {
		statusCode = recorder.StatusCode()
		writtenBytes = recorder.WrittenBytes()
	}

	timingResults := Fields{}
	fields["timings"] = timingResults
	timingResults["total"] = roundMS(serveDone.Sub(startTime))
	fields["status"] = statusCode
	requestFields["status"] = statusCode

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, rw.Header()),
	}
	fields["response"] = responseFields

	if writtenBytes > 0 {
		responseFields["bytes"] = writtenBytes
	}

	requestFields["tls"] = req.TLS != nil
	if req.URL.Scheme != "" {
		requestFields["proto"] = req.URL.Scheme
	} else if req.TLS != nil && req.TLS.HandshakeComplete {
		requestFields["proto"] = "https"
	}

	if fields["port"] == "" {
		if requestFields["proto"] == "https" {
			fields["port"] = "443"
		} else {
			fields["port"] = "80"
		}
	}

	fields["url"] = requestFields["proto"].(string) + "://" + req.URL.Host + path.String()

	var err errors.GoError
	fields["client_ip"], _ = splitHostPort(req.RemoteAddr)

	if ctxErr, ok := req.Context().Value(request.Error).(errors.GoError); ok {
		err = ctxErr
	}

	entry := log.logger.WithFields(logrus.Fields(fields))
	entry.Time = startTime

	if err != nil {
		entry.WithError(err).Error()
	} else {
		entry.Info()
	}
}
