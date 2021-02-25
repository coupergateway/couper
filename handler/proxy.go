package handler

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/utils"
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
		//return nil, err \\ TODO: log
	}

	var path string
	if pathMatch, ok := req.Context().
		Value(request.Wildcard).(string); ok && strings.HasSuffix(path, "/**") {
		if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
			pathMatch += "/"
		}

		req.URL.Path = utils.JoinPath("/", strings.ReplaceAll(path, "/**", "/"), pathMatch)
	} else if path != "" {
		req.URL.Path = utils.JoinPath("/", path)
	}

	// TODO: call reverseProxy with rec !
	beresp, err := p.backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	_ = eval.ApplyResponseContext(p.evalCtx, p.context, req, beresp) // TODO: log
	return beresp, err
}
