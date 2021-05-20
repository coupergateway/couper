package handler

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
)

// headerBlacklist lists all header keys which will be removed after
// context variable evaluation to ensure to not pass them upstream.
var headerBlacklist = []string{"Authorization", "Cookie"}

// Proxy wraps a httputil.ReverseProxy to apply additional configuration context
// and have control over the roundtrip configuration.
type Proxy struct {
	backend      http.RoundTripper
	context      hcl.Body
	logger       *logrus.Entry
	reverseProxy *httputil.ReverseProxy
}

func NewProxy(backend http.RoundTripper, ctx hcl.Body, logger *logrus.Entry) *Proxy {
	proxy := &Proxy{
		backend: backend,
		context: ctx,
		logger:  logger,
	}
	rp := &httputil.ReverseProxy{
		Director: proxy.director,
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, err error) {
			if rec, ok := rw.(*transport.Recorder); ok {
				rec.SetError(err)
			}
		},
		ErrorLog:  newErrorLogWrapper(logger),
		Transport: backend,
	}
	proxy.reverseProxy = rp
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := eval.ApplyRequestContext(req.Context(), p.context, req); err != nil {
		return nil, err
	}

	url, err := eval.GetContextAttribute(p.context, req.Context(), "url")
	if err != nil {
		return nil, err
	}
	if url != "" {
		ctx := context.WithValue(req.Context(), request.URLAttribute, url)
		*req = *req.WithContext(ctx)
	}

	rec := transport.NewRecorder()
	p.reverseProxy.ServeHTTP(rec, req)
	beresp, err := rec.Response(req)
	if err != nil {
		return beresp, err
	}
	err = eval.ApplyResponseContext(req.Context(), p.context, beresp) // TODO: log only
	return beresp, err
}

func (p *Proxy) director(req *http.Request) {
	for _, key := range headerBlacklist {
		req.Header.Del(key)
	}
}

// ErrorWrapper logs httputil.ReverseProxy internals with our own logrus.Entry.
type ErrorWrapper struct{ l logrus.FieldLogger }

func (e *ErrorWrapper) Write(p []byte) (n int, err error) {
	e.l.Error(strings.Replace(string(p), "\n", "", 1))
	return len(p), nil
}
func newErrorLogWrapper(logger logrus.FieldLogger) *log.Logger {
	return log.New(&ErrorWrapper{logger}, "", log.Lshortfile)
}
