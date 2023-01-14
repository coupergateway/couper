package logging

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
	"github.com/hashicorp/hcl/v2"
)

var (
	_ http.RoundTripper = &UpstreamLog{}
	_ seetie.Object     = &UpstreamLog{}
)

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

	if depOn, ok := req.Context().Value(request.EndpointSequenceDependsOn).(string); ok && depOn != "" {
		fields["depends_on"] = depOn
	}

	if u.config.TypeFieldKey != "" {
		fields["type"] = u.config.TypeFieldKey
	}

	requestFields := Fields{
		"name":   req.Context().Value(request.RoundTripName),
		"method": req.Method,
	}

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	if !u.config.NoProxyFromEnv {
		proxyURL, perr := http.ProxyFromEnvironment(req)
		if perr == nil && proxyURL != nil {
			fields["proxy"] = proxyURL.Host
		}
	}

	fields["request"] = requestFields

	berespBytes := int64(0)
	logCtxCh := make(chan hcl.Body, 17) // TODO: Will block with oauth2 token retries >= 17
	tokenRetries := uint8(0)
	outctx := context.WithValue(req.Context(), request.LogCustomUpstream, logCtxCh)
	outctx = context.WithValue(outctx, request.BackendBytes, &berespBytes)
	outctx = context.WithValue(outctx, request.TokenRequestRetries, &tokenRetries)
	oCtx, openAPIContext := validation.NewWithContext(outctx)
	outreq := req.WithContext(httptrace.WithClientTrace(oCtx, clientTrace))

	rtStart := time.Now()
	beresp, err := u.next.RoundTrip(outreq)
	rtDone := time.Now()

	close(logCtxCh)

	// FIXME: Can host be empty?
	if outreq.Host != "" {
		requestFields["origin"] = outreq.Host
		requestFields["host"], requestFields["port"] = splitHostPort(outreq.Host)
		if requestFields["port"] == "" {
			delete(requestFields, "port")
		}
	}

	requestFields["proto"] = outreq.URL.Scheme

	path := &url.URL{
		Path:       outreq.URL.Path,
		RawPath:    outreq.URL.RawPath,
		RawQuery:   outreq.URL.RawQuery,
		ForceQuery: outreq.URL.ForceQuery,
		Fragment:   outreq.URL.Fragment,
	}
	requestFields["path"] = path.String()
	requestFields["headers"] = filterHeader(u.config.RequestHeaders, outreq.Header)

	fields["url"] = outreq.URL.String()

	if outreq.URL.User != nil && outreq.URL.User.Username() != "" {
		fields["auth_user"] = outreq.URL.User.Username()
	} else if user, _, ok := outreq.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	if tr, ok := outreq.Context().Value(request.TokenRequest).(string); ok && tr != "" {
		fields["token_request"] = tr

		if tokenRetries > 0 {
			fields["token_request_retry"] = tokenRetries
		}
	}

	fields["status"] = 0
	if beresp != nil {
		fields["status"] = beresp.StatusCode
		cl := int64(0)
		if beresp.ContentLength > 0 {
			cl = beresp.ContentLength
		}
		responseFields := Fields{
			"bytes":   cl,
			"headers": filterHeader(u.config.ResponseHeaders, beresp.Header),
			"status":  beresp.StatusCode,
		}
		fields["response"] = responseFields
	}

	if validationErrors := openAPIContext.Errors(); len(validationErrors) > 0 {
		fields["validation"] = validationErrors
	}

	serverTimingsVal := make(utils.ServerTimings)
	serverTimingsKey := ""

	// For test cases: "backend" is sometimes not set
	if u.log != nil {
		if key, exists := u.log.Data["backend"]; exists {
			serverTimingsKey = key.(string)
		}
	}

	if name, ok := requestFields["name"].(string); ok && name != config.DefaultNameLabel {
		serverTimingsKey += "_" + name
	}

	timingResults := Fields{
		"total": RoundMS(rtDone.Sub(rtStart)),
	}
	timingsMu.RLock()
	for f, v := range timings { // clone
		timingResults[f] = v

		serverTimingsVal[fmt.Sprintf(`%s_%s`, serverTimingsKey, f)] = fmt.Sprintf(`dur=%.3f`, v)
	}
	timingsMu.RUnlock()
	fields["timings"] = timingResults
	//timings["ttlb"] = RoundMS(rtDone.Sub(timeTTFB)) // TODO: depends on stream or buffer

	if beresp != nil {
		beresp.Request = outreq.WithContext(context.WithValue(outreq.Context(), request.ServerTimings, serverTimingsVal))
	}

	entry := u.log.
		WithFields(logrus.Fields(fields)).
		WithContext(outreq.Context()).
		WithTime(startTime)

	stack, stacked := FromContext(outreq.Context())

	if err != nil {
		if _, ok := err.(errors.GoError); !ok {
			err = errors.Backend.With(err)
		}
		entry = entry.WithError(err)
		if stacked {
			stack.Push(entry).Level(logrus.ErrorLevel)
		} else {
			entry.Error()
		}
	} else {
		if stacked {
			stack.Push(entry).Level(logrus.InfoLevel)
		} else {
			entry.Info()
		}
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
			timings["ttfb"] = RoundMS(timeTTFB.Sub(timeGotConn))
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
				timings["tcp"] = RoundMS(time.Since(timeConnect))
				mapMu.Unlock()
			}
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			mapMu.Lock()
			timings["dns"] = RoundMS(time.Since(timeDNS))
			mapMu.Unlock()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				mapMu.Lock()
				timings["tls"] = RoundMS(time.Since(timeTLS))
				mapMu.Unlock()
			}
		},
	}

	return timings, mapMu, trace
}

func (u *UpstreamLog) Value() cty.Value {
	next, ok := u.next.(seetie.Object)
	if !ok {
		return cty.NilVal
	}

	return next.Value()
}
