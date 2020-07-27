package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ http.Handler = &Proxy{}

type Proxy struct {
	originURL              *url.URL
	origin, hostname, path string
	evalContext            *hcl.EvalContext
	contextOptions         hcl.Body
	rp                     *httputil.ReverseProxy
	log                    *logrus.Entry
}

func NewProxy(origin, hostname, path string, log *logrus.Entry, evalCtx *hcl.EvalContext, options hcl.Body) http.Handler {
	originURL, err := url.Parse(origin)
	if err != nil {
		panic("err parsing origin url: " + err.Error())
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		panic("err: backend origin must define a scheme")
	}

	proxy := &Proxy{
		origin:         origin,
		originURL:      originURL,
		evalContext:    evalCtx,
		hostname:       hostname,
		path:           path,
		log:            log,
		contextOptions: options,
	}

	proxy.rp = &httputil.ReverseProxy{
		Director:       proxy.director, // request modification
		ModifyResponse: proxy.modifyResponse,
	}
	return proxy
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.rp.ServeHTTP(rw, req)
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.originURL.Host
	req.URL.Scheme = p.originURL.Scheme
	req.Host = p.originURL.Host
	if p.hostname != "" {
		req.Host = p.hostname
	}

	if pathMatch, ok := req.Context().Value("route_wildcard").(string); ok && p.path != "" {
		req.URL.Path = path.Join(strings.ReplaceAll(p.path, "/**", "/"), pathMatch)
	} else if p.path != "" {
		req.URL.Path = p.path
	}

	log := p.log.WithField("uid", req.Context().Value("requestID"))

	contextOptions, err := NewRequestCtxOptions(p.evalContext, p.contextOptions, req)
	if err != nil {
		log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
		return
	}

	if contextOptions.Request == nil {
		return
	}
	for header, value := range contextOptions.Request.Headers {
		if len(value) == 0 {
			req.Header.Del(header)
		} else {
			req.Header.Set(header, value[0])
		}
	}
	if len(contextOptions.Request.Headers) > 0 {
		log.WithField("custom-req-header", contextOptions.Request.Headers).Debug()
	}
}

func (p *Proxy) modifyResponse(res *http.Response) error {
	log := p.log.WithField("uid", res.Request.Context().Value("requestID"))
	contextOptions, err := NewResponseCtxOptions(p.evalContext, p.contextOptions, res)
	if err != nil {
		log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
		return err
	}

	if contextOptions.Response == nil {
		return nil
	}

	for header, value := range contextOptions.Response.Headers {
		if len(value) == 0 {
			res.Header.Del(header)
		} else {
			res.Header.Set(header, value[0])
		}
	}
	if len(contextOptions.Response.Headers) > 0 {
		log.WithField("custom-res-header", contextOptions.Response.Headers).Debug()
	}
	return nil
}

func (p *Proxy) String() string {
	return "Proxy"
}
