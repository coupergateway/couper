package backend

import (
	"net/http"
	"net/http/httputil"
	"path"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var (
	_ http.Handler = &Proxy{}
)

type Proxy struct {
	OriginAddress  string   `hcl:"origin_address"`
	OriginHost     string   `hcl:"origin_host"`
	OriginScheme   string   `hcl:"origin_scheme,optional"` // optional defaults to attr
	Path           string   `hcl:"path,optional"`
	ContextOptions hcl.Body `hcl:",remain"`
	rp             *httputil.ReverseProxy
	log            *logrus.Entry
}

func NewProxy() func(*logrus.Entry) http.Handler {
	return func(log *logrus.Entry) http.Handler {
		proxy := &Proxy{log: log}
		proxy.rp = &httputil.ReverseProxy{
			Director:       proxy.director, // request modification
			ModifyResponse: proxy.modifyResponse,
		}
		return proxy
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.rp.ServeHTTP(rw, req)
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.OriginAddress
	req.URL.Scheme = "http"
	if strings.HasSuffix(p.OriginAddress, "443") {
		req.URL.Scheme = "https" // TODO: improve conf options, scheme or url
	}
	req.Host = p.OriginHost
	if p.Path != "" {
		req.URL.Path = path.Join("/", p.Path)
	}

	log := p.log.WithField("uid", req.Context().Value("requestID"))
	contextOptions, err := NewRequestCtxOptions(p.ContextOptions, req)
	if err != nil {
		log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
		return
	}

	for header, value := range contextOptions.Request.Headers {
		req.Header.Set(header, value[0])
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

	for header, value := range contextOptions.Response.Headers {
		res.Header.Set(header, value[0])
	}
	if len(contextOptions.Request.Headers) > 0 {
		log.WithField("custom-res-header", contextOptions.Response.Headers).Debug()
	}
	return nil
}

func (p *Proxy) String() string {
	return "Proxy"
}
