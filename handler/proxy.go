package handler

import (
	"net/http"
	"net/http/httputil"

	"github.com/hashicorp/hcl/v2"

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
	reverseProxy *httputil.ReverseProxy
}

func NewProxy(backend http.RoundTripper, ctx hcl.Body) *Proxy {
	proxy := &Proxy{
		backend: backend,
		context: ctx,
	}
	rp := &httputil.ReverseProxy{
		Director:  proxy.director,
		Transport: backend,
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, err error) {
			if rec, ok := rw.(*transport.Recorder); ok {
				rec.SetError(err)
			}
		},
	}
	proxy.reverseProxy = rp
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := eval.ApplyRequestContext(req.Context(), p.context, req); err != nil {
		return nil, err // TODO: log only
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
