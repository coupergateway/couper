package logging

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

type RoundtripInfo struct {
	BeReq  *http.Request
	BeResp *http.Response
	Err    error
}

type RoundtripHandlerFunc http.HandlerFunc

func (f RoundtripHandlerFunc) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f(rw, req)
}

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

var handlerFuncType = reflect.ValueOf(RoundtripHandlerFunc(nil)).Type()

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler, startTime time.Time) {
	handlerType := reflect.ValueOf(nextHandler).Type()
	isUpstreamRequest := handlerType == handlerFuncType

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

	ctx := httptrace.WithClientTrace(req.Context(), trace)
	var roundtripInfo *RoundtripInfo
	if isUpstreamRequest {
		roundtripInfo = &RoundtripInfo{}
		ctx = context.WithValue(ctx, request.RoundtripInfo, roundtripInfo)
	}
	*req = *req.Clone(ctx)

	statusRecorder := NewStatusRecorder(rw)
	rw = statusRecorder

	nextHandler.ServeHTTP(rw, req)
	serveDone := time.Now()

	reqCtx := req
	if isUpstreamRequest && roundtripInfo != nil && roundtripInfo.BeReq != nil {
		reqCtx = roundtripInfo.BeReq
		if roundtripInfo.BeResp != nil {
			reqCtx.TLS = roundtripInfo.BeResp.TLS
		}
	}

	uniqueID := reqCtx.Context().Value(request.UID)

	requestFields := Fields{
		"headers": filterHeader(log.conf.RequestHeaders, reqCtx.Header),
	}

	if req.ContentLength > 0 {
		requestFields["bytes"] = reqCtx.ContentLength
	}

	serverName, _ := reqCtx.Context().Value(request.ServerName).(string)

	fields := Fields{
		"method":  reqCtx.Method,
		"proto":   reqCtx.Proto,
		"request": requestFields,
		"server":  serverName,
		"uid":     uniqueID,
	}

	if h, ok := nextHandler.(fmt.Stringer); ok {
		fields["handler"] = h.String()
	}

	if isUpstreamRequest {
		backendName, _ := reqCtx.Context().Value(request.BackendName).(string)
		if backendName == "" {
			endpointName, _ := reqCtx.Context().Value(request.Endpoint).(string)
			backendName = serverName + ":" + endpointName
		}
		fields["backend"] = backendName
	}

	if log.conf.TypeFieldKey != "" {
		fields["type"] = log.conf.TypeFieldKey
	}

	path := &url.URL{
		Path:       reqCtx.URL.Path,
		RawPath:    reqCtx.URL.RawPath,
		RawQuery:   reqCtx.URL.RawQuery,
		ForceQuery: reqCtx.URL.ForceQuery,
		Fragment:   reqCtx.URL.Fragment,
	}
	requestFields["path"] = path.String()

	if req.Host != "" {
		requestFields["addr"] = reqCtx.Host
		requestFields["host"], requestFields["port"] = splitHostPort(reqCtx.Host)
	}

	if reqCtx.URL.User != nil && reqCtx.URL.User.Username() != "" {
		fields["auth_user"] = reqCtx.URL.User.Username()
	} else if user, _, ok := reqCtx.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	fields["realtime"] = roundMS(serveDone.Sub(startTime))
	fields["status"] = statusRecorder.status

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, rw.Header()),
	}
	fields["response"] = responseFields

	if statusRecorder.writtenBytes > 0 {
		responseFields["bytes"] = statusRecorder.writtenBytes
	}

	scheme := "http"
	if reqCtx.TLS != nil {
		scheme = "https"
	}
	fields["scheme"] = scheme

	var err error
	if isUpstreamRequest && roundtripInfo != nil {
		err = roundtripInfo.Err
		fields["status"] = 0
		if roundtripInfo.BeResp != nil {
			fields["timings"] = timings
			fields["status"] = roundtripInfo.BeResp.StatusCode
			timings["ttlb"] = roundMS(serveDone.Sub(timeTTFB))
		} else {
			fields["scheme"] = reqCtx.URL.Scheme
		}
	} else if !isUpstreamRequest {
		fields["client_ip"], _ = splitHostPort(reqCtx.RemoteAddr)
		if couperErr := statusRecorder.Header().Get(errors.HeaderErrorCode); couperErr != "" {
			i, _ := strconv.Atoi(couperErr[:4])
			err = errors.Code(i)
			fields["code"] = i
		}
	}

	var entry *logrus.Entry
	if log.conf.ParentFieldKey != "" {
		entry = log.logger.WithField(log.conf.ParentFieldKey, fields)
	} else {
		entry = log.logger.WithFields(logrus.Fields(fields))
	}
	entry.Time = startTime

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

func roundMS(d time.Duration) string {
	const maxDuration time.Duration = 1<<63 - 1
	if d == maxDuration {
		return "0.0"
	}
	return fmt.Sprintf("%.3f", math.Round(float64(d)*1000)/1000/float64(time.Millisecond))
}
