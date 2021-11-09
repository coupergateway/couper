package logging

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/validation"
)

var _ http.RoundTripper = &UpstreamLog{}

type UpstreamLog struct {
	config *Config
	log    *logrus.Entry
	next   http.RoundTripper
}

func NewUpstreamLog(log *logrus.Entry, next http.RoundTripper, ignoreProxyEnv bool) *UpstreamLog {
	logConf := *DefaultConfig
	logConf.NoProxyFromEnv = ignoreProxyEnv
	logConf.TypeFieldKey = "couper_backend"
	env.DecodeWithPrefix(&logConf, "BACKEND_")
	return &UpstreamLog{
		config: &logConf,
		log:    log,
		next:   next,
	}
}

func (u *UpstreamLog) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	timings, timingsMu, clientTrace := u.newTraceContext()

	fields := Fields{
		"uid":    req.Context().Value(request.UID),
		"method": req.Method,
	}

	if u.config.TypeFieldKey != "" {
		fields["type"] = u.config.TypeFieldKey
	}

	requestFields := Fields{
		"name":   req.Context().Value(request.RoundTripName),
		"method": req.Method,
		"proto":  req.URL.Scheme,
	}

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	if !u.config.NoProxyFromEnv {
		proxyUrl, perr := http.ProxyFromEnvironment(req)
		if perr == nil && proxyUrl != nil {
			fields["proxy"] = proxyUrl.Host
		}
	}

	fields["request"] = requestFields

	oCtx, openAPIContext := validation.NewWithContext(req.Context())
	*req = *req.WithContext(httptrace.WithClientTrace(oCtx, clientTrace))

	rtStart := time.Now()
	beresp, err := u.next.RoundTrip(req)
	rtDone := time.Now()

	if customLogs, ok := req.Context().Value(request.BackendLogFields).(logrus.Fields); ok && len(customLogs) > 0 {
		fields["custom"] = customLogs
	}

	if req.Host != "" {
		requestFields["origin"] = req.Host
		requestFields["host"], requestFields["port"] = splitHostPort(req.Host)
		if requestFields["port"] == "" {
			delete(requestFields, "port")
		}
	}

	path := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	requestFields["path"] = path.String()
	requestFields["headers"] = filterHeader(u.config.RequestHeaders, req.Header)

	fields["url"] = req.URL.String()

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	if tr, ok := req.Context().Value(request.TokenRequest).(string); ok && tr != "" {
		fields["token_request"] = tr

		if retries, ok := req.Context().Value(request.TokenRequestRetries).(uint8); ok && retries > 0 {
			fields["token_request_retry"] = retries
		}
	}

	if opt, ok := req.Context().Value(request.BufferOptions).(eval.BufferOption); ok {
		fields["buffered"] = opt.GoString()
	}

	fields["status"] = 0
	if beresp != nil {
		fields["status"] = beresp.StatusCode
		responseFields := Fields{
			"headers": filterHeader(u.config.ResponseHeaders, beresp.Header),
			"status":  beresp.StatusCode,
		}
		fields["response"] = responseFields
	}

	if validationErrors := openAPIContext.Errors(); len(validationErrors) > 0 {
		fields["validation"] = validationErrors
	}

	timingResults := Fields{
		"total": roundMS(rtDone.Sub(rtStart)),
	}
	timingsMu.RLock()
	for f, v := range timings { // clone
		timingResults[f] = v
	}
	timingsMu.RUnlock()
	fields["timings"] = timingResults
	//timings["ttlb"] = roundMS(rtDone.Sub(timeTTFB)) // TODO: depends on stream or buffer

	entry := u.log.WithFields(logrus.Fields(fields))
	entry.Time = startTime

	if err != nil {
		if _, ok := err.(errors.GoError); !ok {
			err = errors.Backend.With(err)
		}
		entry.WithError(err).Error()
	} else {
		entry.Info()
	}

	return beresp, err
}

func (u *UpstreamLog) LogEntry() *logrus.Entry {
	// TODO: field enrichment / copy
	// used for validation errors
	return u.log
}

func (u *UpstreamLog) newTraceContext() (Fields, *sync.RWMutex, *httptrace.ClientTrace) {
	timings := Fields{}
	mapMu := &sync.RWMutex{}
	var timeTTFB, timeGotConn, timeConnect, timeDNS, timeTLS time.Time
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			now := time.Now()
			mapMu.Lock()
			timeGotConn = now
			mapMu.Unlock()
		},
		GotFirstResponseByte: func() {
			timeTTFB = time.Now()
			mapMu.Lock()
			timings["ttfb"] = roundMS(timeTTFB.Sub(timeGotConn))
			mapMu.Unlock()
		},
		ConnectStart: func(_, _ string) {
			now := time.Now()
			mapMu.Lock()
			timeConnect = now
			mapMu.Unlock()
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			now := time.Now()
			mapMu.Lock()
			timeDNS = now
			mapMu.Unlock()
		},
		TLSHandshakeStart: func() {
			now := time.Now()
			mapMu.Lock()
			timeTLS = now
			mapMu.Unlock()
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				mapMu.Lock()
				timings["tcp"] = roundMS(time.Since(timeConnect))
				mapMu.Unlock()
			}
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			mapMu.Lock()
			timings["dns"] = roundMS(time.Since(timeDNS))
			mapMu.Unlock()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				mapMu.Lock()
				timings["tls"] = roundMS(time.Since(timeTLS))
				mapMu.Unlock()
			}
		},
	}

	return timings, mapMu, trace
}
