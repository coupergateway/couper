package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ http.Handler = &Proxy{}

type Proxy struct {
	evalContext *hcl.EvalContext
	log         *logrus.Entry
	options     *ProxyOptions
	originURL   *url.URL
	rp          *httputil.ReverseProxy
}

type ProxyOptions struct {
	ConnectTimeout, Timeout time.Duration
	Context                 hcl.Body
	Hostname, Origin, Path  string
}

func NewProxy(options *ProxyOptions, log *logrus.Entry, evalCtx *hcl.EvalContext) http.Handler {
	originURL, err := url.Parse(options.Origin)
	if err != nil {
		panic("err parsing origin url: " + err.Error())
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		panic("err: backend origin must define a scheme")
	}

	proxy := &Proxy{
		evalContext: evalCtx,
		log:         log,
		options:     options,
		originURL:   originURL,
	}

	d := &net.Dialer{Timeout: options.ConnectTimeout}
	proxy.rp = &httputil.ReverseProxy{
		Director: proxy.director, // request modification
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) { // TODO: merge with error logging
			rw.WriteHeader(http.StatusBadGateway)
			log.WithField("uid", req.Context().Value("requestID")).Error(err)
		},
		ModifyResponse: proxy.modifyResponse,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s failed: %w", originURL.String(), err)
				}
				return conn, nil
			},
		},
	}
	return proxy
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	deadline := time.Now().Add(p.options.Timeout)
	ctx, cancelFn := context.WithDeadline(req.Context(), deadline)
	defer cancelFn()
	p.rp.ServeHTTP(rw, req.WithContext(ctx))
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.originURL.Host
	req.URL.Scheme = p.originURL.Scheme
	req.Host = p.originURL.Host
	if p.options.Hostname != "" {
		req.Host = p.options.Hostname
	}

	if pathMatch, ok := req.Context().Value("route_wildcard").(string); ok && p.options.Path != "" {
		req.URL.Path = path.Join(strings.ReplaceAll(p.options.Path, "/**", "/"), pathMatch)
	} else if p.options.Path != "" {
		req.URL.Path = p.options.Path
	}

	log := p.log.WithField("uid", req.Context().Value("requestID"))

	contextOptions, err := NewRequestCtxOptions(p.evalContext, p.options.Context, req)
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
	contextOptions, err := NewResponseCtxOptions(p.evalContext, p.options.Context, res)
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
