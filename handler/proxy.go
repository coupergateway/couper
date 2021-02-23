package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
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
		Director:  proxy.director,
		Transport: proxy,
	}
	proxy.reverseProxy = rp
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "" || req.URL.Scheme == "" {
		return nil, errors.New("proxy: origin not set")
	}

	if err := eval.ApplyRequestContext(p.evalCtx, p.context, req); err != nil {
		return nil, err
	}
	beresp, err := p.backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	err = eval.ApplyResponseContext(p.evalCtx, p.context, req, beresp)
	return beresp, err
}

var backendInlineSchema = config.Backend{}.Schema(true)

func (p *Proxy) director(req *http.Request) {
	var origin, hostname, path string
	// TODO: apply eval via produce // bufferopts
	httpContext := eval.NewHTTPContext(p.evalCtx, eval.BufferNone, req, nil, nil)
	content, _, _ := p.context.PartialContent(backendInlineSchema)
	if o := getAttribute(httpContext, "origin", content); o != "" {
		origin = o
	}
	if h := getAttribute(httpContext, "hostname", content); h != "" {
		hostname = h
	}
	if pathVal := getAttribute(httpContext, "path", content); pathVal != "" {
		path = pathVal
	}

	originURL, _ := url.Parse(origin)

	req.URL.Host = originURL.Host
	req.URL.Scheme = originURL.Scheme
	req.Host = originURL.Host

	if hostname != "" {
		req.Host = hostname
	}

	if pathMatch, ok := req.Context().
		Value(request.Wildcard).(string); ok && strings.HasSuffix(path, "/**") {
		if strings.HasSuffix(req.URL.Path, "/") && !strings.HasSuffix(pathMatch, "/") {
			pathMatch += "/"
		}

		req.URL.Path = utils.JoinPath("/", strings.ReplaceAll(path, "/**", "/"), pathMatch)
	} else if path != "" {
		req.URL.Path = utils.JoinPath("/", path)
	}
}

func getAttribute(ctx *hcl.EvalContext, name string, body *hcl.BodyContent) string {
	attr := body.Attributes
	if _, ok := attr[name]; !ok {
		return ""
	}
	originValue, _ := attr[name].Expr.Value(ctx)
	return seetie.ValueToString(originValue)
}
