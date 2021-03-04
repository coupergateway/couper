package logging

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
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
	logConf.TypeFieldKey = "couper_upstream"
	env.DecodeWithPrefix(&logConf, "UPSTREAM_")
	return &UpstreamLog{
		config: &logConf,
		log:    log,
		next:   next,
	}
}

func (u *UpstreamLog) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	timings := u.withTraceContext(req)

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
		"headers": filterHeader(u.config.RequestHeaders, req.Header),
		"method":  req.Method,
		"name":    req.Context().Value(request.RoundTripName),
		"proto":   req.Proto,
		"scheme":  req.URL.Scheme,
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

	if req.Host != "" {
		requestFields["addr"] = req.Host
		requestFields["host"], requestFields["port"] = splitHostPort(req.Host)
	}

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

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
		//timings["ttlb"] = roundMS(rtDone.Sub(timeTTFB)) // TODO: depends on stream or buffer

		if couperErr := beresp.Header.Get(errors.HeaderErrorCode); couperErr != "" {
			i, _ := strconv.Atoi(couperErr[:4])
			err = errors.Code(i) // TODO: override original one??
			fields["code"] = i
		}
	}

	if validationErrors := openAPIContext.Errors(); len(validationErrors) > 0 {
		fields["validation"] = validationErrors
	}

	fields["timings"] = timings
	var entry *logrus.Entry
	if u.config.ParentFieldKey != "" {
		entry = u.log.WithField(u.config.ParentFieldKey, fields)
	} else {
		entry = u.log.WithFields(logrus.Fields(fields))
	}
	entry.Time = startTime

	if (beresp != nil && beresp.StatusCode == http.StatusInternalServerError) || err != nil {
		if err != nil {
			entry.Error(err)
			return beresp, err
		}
		entry.Error()
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

func (u *UpstreamLog) withTraceContext(req *http.Request) Fields {
	timings := Fields{}
	var timeTTFB, timeGotConn, timeConnect, timeDNS, timeTLS time.Time
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			timeGotConn = time.Now()
		},
		GotFirstResponseByte: func() {
			timeTTFB = time.Now()
			timings["ttfb"] = roundMS(timeTTFB.Sub(timeGotConn))
		},
		ConnectStart: func(_, _ string) {
			timeConnect = time.Now()
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			timeDNS = time.Now()
		},
		TLSHandshakeStart: func() {
			timeTLS = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				timings["connect"] = roundMS(time.Since(timeConnect))
			}
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			timings["dns"] = roundMS(time.Since(timeDNS))
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				timings["tls"] = roundMS(time.Since(timeTLS))
			}
		},
	}
	ctx := httptrace.WithClientTrace(req.Context(), trace)
	*req = *req.WithContext(ctx)
	return timings
}
