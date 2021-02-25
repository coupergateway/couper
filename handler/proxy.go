package handler

import (
	"net/http"
	"net/http/httputil"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/eval"
)

// Proxy wraps a httputil.ReverseProxy to apply additional configuration context
// and have control over the roundtrip configuration.
type Proxy struct {
	backend      http.RoundTripper
	context      hcl.Body
	evalCtx      *hcl.EvalContext
	reverseProxy *httputil.ReverseProxy
}

func NewProxy(backend http.RoundTripper, ctx hcl.Body, evalCtx *hcl.EvalContext) *Proxy {
	proxy := &Proxy{
		backend: backend,
		context: ctx,
		evalCtx: evalCtx,
	}
	rp := &httputil.ReverseProxy{
		//Director:  proxy.director,
		Transport: proxy,
	}
	proxy.reverseProxy = rp
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := eval.ApplyRequestContext(p.evalCtx, p.context, req); err != nil {
		return nil, err // TODO: log only
	}

	// TODO: call reverseProxy with rec !
	beresp, err := p.backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	err = eval.ApplyResponseContext(p.evalCtx, p.context, req, beresp) // TODO: log only
	return beresp, err
}
