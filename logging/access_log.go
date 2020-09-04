package logging

import (
	"crypto/tls"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"reflect"
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

var handlerFuncType = reflect.ValueOf(http.HandlerFunc(nil)).Type()

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler) {
	startTime := time.Now()

	isUpstreamRequest := reflect.ValueOf(nextHandler).Type() == handlerFuncType

	clientAddr := req.RemoteAddr
	timings := Fields{}
	var timeTTFB, timeGotConn time.Time
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			timeGotConn = time.Now()
		},
		GotFirstResponseByte: func() {
			timeTTFB = time.Now()
			if isUpstreamRequest {
				timings["ttfb"] = roundMS(timeTTFB.Sub(timeGotConn))
			}
		},
	}

	if isUpstreamRequest {
		var timeConnect, timeDNS, timeTLS time.Time
		trace.ConnectStart = func(_, _ string) {
			timeConnect = time.Now()
		}
		trace.DNSStart = func(_ httptrace.DNSStartInfo) {
			timeDNS = time.Now()
		}
		trace.TLSHandshakeStart = func() {
			timeTLS = time.Now()
		}
		trace.ConnectDone = func(network, addr string, err error) {
			if err == nil {
				timings["connect"] = roundMS(time.Since(timeConnect))
			}
		}
		trace.DNSDone = func(_ httptrace.DNSDoneInfo) {
			timings["dns"] = roundMS(time.Since(timeDNS))
		}
		trace.TLSHandshakeDone = func(_ tls.ConnectionState, err error) {
			if err == nil {
				timings["tls"] = roundMS(time.Since(timeTLS))
			}
		}
	}

	*req = *req.Clone(httptrace.WithClientTrace(req.Context(), trace))

	statusRecorder := NewStatusRecorder(rw)
	rw = statusRecorder

	uniqueID := req.Context().Value(request.UID)
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
		"timestamp":         startTime.UTC(),
		"uid":               uniqueID,
		"timings":           timings,
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
		"addr": clientAddr,
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
	serveDone := time.Now()

	timings["ttlb"] = roundMS(serveDone.Sub(timeTTFB))

	fields["realtime"] = roundMS(serveDone.Sub(startTime))
	fields["status"] = statusRecorder.status

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, rw.Header()),
	}
	fields["response"] = responseFields

	if statusRecorder.writtenBytes > 0 {
		responseFields["bytes"] = statusRecorder.writtenBytes
	}

	var err error
	if reqErr, ok := req.Context().Value(request.Error).(error); ok {
		err = reqErr
		if isUpstreamRequest && statusRecorder.status == http.StatusBadGateway {
			fields["status"] = 0
		}
	}

	var entry *logrus.Entry
	if log.conf.ParentFieldKey != "" {
		entry = log.logger.WithField(log.conf.ParentFieldKey, fields)
	} else {
		entry = log.logger.WithFields(logrus.Fields(fields))
	}
	if statusRecorder.status == http.StatusInternalServerError || err != nil {
		if err != nil {
			entry.Error(err)
			return
		}
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

func roundMS(d time.Duration) float64 {
	const maxDuration time.Duration = 1<<63 - 1
	if d == maxDuration {
		return 0.0
	}
	return math.Round(float64(d/time.Millisecond*1000)) / 1000
}
