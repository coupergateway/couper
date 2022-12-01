package logging

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"
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

func (log *AccessLog) Do(writer http.ResponseWriter, req *http.Request) {
	serveDone := time.Now()
	startTime := req.Context().Value(request.StartTime).(time.Time)
	fields := Fields{}

	endpointName, _ := req.Context().Value(request.Endpoint).(string)
	if endpointName != "" {
		fields["endpoint"] = endpointName
	}

	fields["method"] = req.Method
	if server, ok := req.Context().Value(request.ServerName).(string); ok && server != "" {
		fields["server"] = server
	}

	if api, ok := req.Context().Value(request.APIName).(string); ok && api != "" {
		fields["api"] = api
	}

	if uid := req.Context().Value(request.UID); uid != nil {
		fields["uid"] = uid
	}

	requestFields := Fields{
		"headers": filterHeader(log.conf.RequestHeaders, req.Header),
		"method":  req.Method,
		"proto":   "https",
	}
	fields["request"] = requestFields

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	// Read out handler kind from context set by muxer
	if handlerName := req.Context().Value(request.Handler); handlerName != nil {
		fields["handler"] = handlerName
	} else if handlerName = req.Context().Value(request.EndpointKind); handlerName != nil {
		fields["handler"] = handlerName // fallback, e.g. with ErrorHandler
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

	if req.URL.Host == "" {
		req.URL.Host = req.Host // fallback e.g. metrics
	}
	requestFields["origin"] = req.URL.Host
	requestFields["host"], fields["port"] = splitHostPort(req.URL.Host)

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	var statusCode int
	var writtenBytes int
	if recorder, ok := writer.(RecorderInfo); ok {
		statusCode = recorder.StatusCode()
		writtenBytes = recorder.WrittenBytes()
	}

	timingResults := Fields{}
	fields["timings"] = timingResults
	timingResults["total"] = RoundMS(serveDone.Sub(startTime))
	fields["status"] = statusCode
	requestFields["status"] = statusCode

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, writer.Header()),
	}
	fields["response"] = responseFields

	if writtenBytes > 0 {
		responseFields["bytes"] = writtenBytes
	}

	requestFields["tls"] = req.TLS != nil
	if req.Proto == "HTTP/2.0" {
		// TODO: any way to obtain the StreamID?
		// https://github.com/golang/net/blob/master/http2/writesched.go
		requestFields["h2"] = true
	}

	if req.TLS != nil && req.TLS.HandshakeComplete {
		requestFields["proto"] = "https"
	} else if req.URL.Scheme != "" {
		requestFields["proto"] = req.URL.Scheme
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

	entry := log.logger.WithFields(logrus.Fields(fields)).WithContext(req.Context())
	entry.Time = startTime

	if err != nil {
		entry.WithError(err).Error()
	} else {
		if l, ok := req.Context().Value(request.LogDebugLevel).(bool); ok && l {
			entry.Debug()
			return
		}
		entry.Info()
	}
}

func UpdateCustomAccessLogContext(req *http.Request, body hcl.Body) {
	bodies := []hcl.Body{body}
	if b := req.Context().Value(request.LogCustomAccess); b != nil {
		previousBodies, _ := b.([]hcl.Body)
		bodies = append(previousBodies, bodies...)
	}
	*req = *req.WithContext(context.WithValue(req.Context(), request.LogCustomAccess, bodies))
}
