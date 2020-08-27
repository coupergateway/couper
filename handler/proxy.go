package handler

import (
	"context"
	"crypto/tls"
	"errors"
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

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/eval"
)

var (
	_ http.Handler = &Proxy{}

	OriginRequiredError = errors.New("origin is required")
	SchemeRequiredError = errors.New("backend origin must define a scheme")

	// headerBlacklist lists all header keys which will be removed after
	// context variable evaluation to ensure to not pass them upstream.
	headerBlacklist = []string{"Authorization", "Cookie"}
)

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

func NewProxy(options *ProxyOptions, log *logrus.Entry, evalCtx *hcl.EvalContext) (http.Handler, error) {
	if options.Origin == "" {
		return nil, OriginRequiredError
	}
	originURL, err := url.Parse(options.Origin)
	if err != nil {
		return nil, fmt.Errorf("err parsing origin url: %w", err)
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return nil, SchemeRequiredError
	}

	proxy := &Proxy{
		evalContext: evalCtx,
		log:         log,
		options:     options,
		originURL:   originURL,
	}

	var tlsConf *tls.Config
	if options.Hostname != "" {
		tlsConf = &tls.Config{
			ServerName: options.Hostname,
		}
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
			TLSClientConfig:       tlsConf,
		},
	}
	return proxy, nil
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if p.options.Timeout > 0 {
		deadline := time.Now().Add(p.options.Timeout)
		c, cancelFn := context.WithDeadline(req.Context(), deadline)
		ctx = c
		defer cancelFn()
	}
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

	if pathMatch, ok := req.Context().Value(config.WildcardCtxKey).(string); ok && p.options.Path != "" {
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

func (p *Proxy) setRoundtripContext(req *http.Request, beresp *http.Response) {
	var reqCtx context.Context
	var attrCtx string
	var headerCtx http.Header
	if req != nil {
		reqCtx = req.Context()
		headerCtx = req.Header
		attrCtx = attrReqHeaders
	} else if beresp != nil {
		reqCtx = beresp.Request.Context()
		headerCtx = beresp.Header
		attrCtx = attrResHeaders
	}
	log := p.log.WithField("uid", reqCtx.Value("requestID"))
	var fields []string

	evalCtx := eval.NewHTTPContext(p.evalContext, req, beresp)

	// Remove blacklisted headers after evaluation to be accessable within our context configuration.
	if attrCtx == attrReqHeaders {
		for _, key := range headerBlacklist {
			headerCtx.Del(key)
		}
	}

	for _, ctxBody := range p.options.Context {
		options, err := NewCtxOptions(attrCtx, evalCtx, ctxBody)
		if err != nil {
			log.WithField("type", "couper_hcl").WithField("parse config", p.String()).Error(err)
			return
		}
		fields = append(fields, setFields(headerCtx, options)...)
	}

	logKey := "custom-req-header"
	if len(fields) > 0 {
		if beresp != nil {
			logKey = "custom-res-header"
		}
		log.WithField(logKey, fields).Debug()
	}
}

func (p *Proxy) String() string {
	return "Proxy"
}

func setFields(header http.Header, options OptionsMap) []string {
	var fields []string
	if len(options) == 0 {
		return fields
	}

	for key, value := range options {
		if len(value) == 0 || value[0] == "" {
			header.Del(key)
			continue
		}
		k := http.CanonicalHeaderKey(key)
		header[k] = value
		fields = append(fields, k+": "+strings.Join(value, ","))
	}
	return fields
}
