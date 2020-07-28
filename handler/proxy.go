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

// headerBlacklist lists all header keys which will be removed after
// context variable evaluation to ensure to not pass them upstream.
var headerBlacklist = []string{"Authentication", "Cookie"}

type Proxy struct {
	evalContext *hcl.EvalContext
	log         *logrus.Entry
	options     *ProxyOptions
	originURL   *url.URL
	rp          *httputil.ReverseProxy
}

type ProxyOptions struct {
	ConnectTimeout, Timeout, TTFBTimeout time.Duration
	Context                              []hcl.Body
	Hostname, Origin, Path               string
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
			// DisableCompression: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, fmt.Errorf("connecting to %s failed: %w", originURL.String(), err)
				}
				return conn, nil
			},
			ResponseHeaderTimeout: proxy.options.TTFBTimeout,
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

	p.setRoundtripContext(req, nil)
}

func (p *Proxy) modifyResponse(res *http.Response) error {
	p.setRoundtripContext(nil, res)
	return nil
}

func (p *Proxy) setRoundtripContext(req *http.Request, res *http.Response) {
	var reqCtx context.Context
	if req != nil {
		reqCtx = req.Context()
	} else if res != nil {
		reqCtx = res.Request.Context()
	}
	log := p.log.WithField("uid", reqCtx.Value("requestID"))
	var fields []string

	evalCtx := NewHTTPEvalContext(p.evalContext, req, res)

	if req != nil {
		for _, key := range headerBlacklist {
			req.Header.Del(key)
		}
	}

	for _, ctx := range p.options.Context {
		ctxHeaders := &ContextOptions{}
		err := NewCtxOptions(ctxHeaders, evalCtx, ctx)
		if err != nil {
			log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
			return
		}
		if req != nil {
			fields = append(fields, setFields(req.Header, ctxHeaders.ReqOptions)...)
		} else if res != nil {
			fields = append(fields, setFields(res.Header, ctxHeaders.RespOptions)...)
		}
	}

	logKey := "custom-req-header"
	if len(fields) > 0 {
		if res != nil {
			logKey = "custom-res-header"
		}
		log.WithField(logKey, fields).Debug()
	}
}

func (p *Proxy) String() string {
	return "Proxy"
}

func setFields(header http.Header, ctx http.Header) []string {
	var fields []string
	if len(ctx) == 0 {
		return fields
	}

	for key, value := range ctx {
		if len(value) == 0 || value[0] == "" {
			header.Del(key)
			continue
		}
		header[http.CanonicalHeaderKey(key)] = value
	}
	return fields
}
