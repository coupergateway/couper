package backend

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"
	"go.avenga.cloud/couper/gateway/utils"
)

var (
	_ http.Handler = &Proxy{}
	_ selectable   = &Proxy{}
)

type Proxy struct {
	OriginAddress  string   `hcl:"origin_address"`
	OriginHost     string   `hcl:"origin_host"`
	OriginScheme   string   `hcl:"origin_scheme,optional"` // optional defaults to attr
	Path           string   `hcl:"path,optional"`
	ContextOptions hcl.Body `hcl:",remain"`
	rp             *httputil.ReverseProxy
	log            *logrus.Entry
	options        hcl.Body
}

func NewProxy() func(*logrus.Entry, hcl.Body) http.Handler {
	return func(log *logrus.Entry, options hcl.Body) http.Handler {
		proxy := &Proxy{log: log, options: options}
		proxy.rp = &httputil.ReverseProxy{
			Director:       proxy.director, // request modification
			ModifyResponse: proxy.modifyResponse,
		}
		return proxy
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	decodeCtx := NewEvalContext(req, nil)
	diags := gohcl.DecodeBody(p.options, decodeCtx, p)
	if diags.HasErrors() {
		p.log.Fatal(diags.Error())
	}

	p.rp.ServeHTTP(rw, req)
}

func (p *Proxy) hasResponse(req *http.Request) bool {
	return true
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.OriginAddress
	req.URL.Scheme = "http"
	if strings.HasSuffix(p.OriginAddress, "443") {
		req.URL.Scheme = "https" // TODO: improve conf options, scheme or url
	}
	req.Host = p.OriginHost
	if pathMatch, ok := req.Context().Value("route_wildcard").(string); ok && p.Path != "" {
		req.URL.Path = utils.JoinPath(strings.ReplaceAll(p.Path, "/**", "/"), pathMatch)
	} else if p.Path != "" {
		req.URL.Path = p.Path
	}

	log := p.log.WithField("uid", req.Context().Value("requestID"))
	contextOptions, err := NewRequestCtxOptions(p.ContextOptions, req)
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
	contextOptions, err := NewResponseCtxOptions(p.ContextOptions, res)
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
