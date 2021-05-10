package logging

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
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

	timings, timingsMu := u.withTraceContext(req)

	fields := Fields{
		"uid": req.Context().Value(request.UID),
	}
	if h, ok := u.next.(fmt.Stringer); ok {
		fields["handler"] = h.String()
	}
	if u.config.TypeFieldKey != "" {
		fields["type"] = u.config.TypeFieldKey
	}

	requestFields := Fields{
		"method": req.Method,
		"name":   req.Context().Value(request.RoundTripName),
	}

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	path := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	requestFields["path"] = path.String()

	if !u.config.NoProxyFromEnv {
		proxyUrl, perr := http.ProxyFromEnvironment(req)
		if perr == nil && proxyUrl != nil {
			fields["proxy"] = proxyUrl.Host
		}
	}

	fields["request"] = requestFields

	oCtx, openAPIContext := validation.NewWithContext(req.Context())
	*req = *req.WithContext(oCtx)

	rtStart := time.Now()
	beresp, err := u.next.RoundTrip(req)
	rtDone := time.Now()

	if req.Host != "" {
		requestFields["addr"] = req.Host
		requestFields["host"], requestFields["port"] = splitHostPort(req.Host)
		if requestFields["port"] == "" {
			delete(requestFields, "port")
		}
	}

	requestFields["headers"] = filterHeader(u.config.RequestHeaders, req.Header)

	if burl, ok := req.Context().Value(request.BackendURL).(string); ok {
		fields["url"] = burl
	}

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}
	requestFields["proto"] = req.Proto
	requestFields["scheme"] = req.URL.Scheme

	if tr, ok := req.Context().Value(request.TokenRequest).(string); ok && tr != "" {
		fields["token_request"] = tr

		if retries, ok := req.Context().Value(request.TokenRequestRetries).(uint8); ok && retries > 0 {
			fields["token_request_retry"] = retries
		}
	}

	fields["realtime"] = roundMS(rtDone.Sub(rtStart))

	fields["status"] = 0
	if beresp != nil {
		fields["status"] = beresp.StatusCode

		responseFields := Fields{
			"headers": filterHeader(u.config.ResponseHeaders, beresp.Header),
			"proto":   beresp.Proto,
			"tls":     beresp.TLS != nil,
		}
		fields["response"] = responseFields
	}

	if validationErrors := openAPIContext.Errors(); len(validationErrors) > 0 {
		fields["validation"] = validationErrors
	}

	timingResults := Fields{}
	timingsMu.RLock()
	for f, v := range timings { // clone
		timingResults[f] = v
	}
	timingsMu.RUnlock()
	fields["timings"] = timingResults
	//timings["ttlb"] = roundMS(rtDone.Sub(timeTTFB)) // TODO: depends on stream or buffer

	entry := u.log.WithFields(logrus.Fields(fields))
	entry.Time = startTime

	if (beresp != nil && beresp.StatusCode == http.StatusInternalServerError) || err != nil {
		if err != nil {
			if _, ok := err.(errors.GoError); !ok {
				err = errors.Backend.With(err)
			}
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

func (u *UpstreamLog) withTraceContext(req *http.Request) (Fields, *sync.RWMutex) {
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
				timings["connect"] = roundMS(time.Since(timeConnect))
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
	ctx := httptrace.WithClientTrace(req.Context(), trace)
	*req = *req.WithContext(ctx)
	return timings, mapMu
}
