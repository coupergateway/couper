package handler

import (
	"bytes"
	"io"
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
	backend          http.RoundTripper
	bufferOption     eval.BufferOption
	context          hcl.Body
	evalCtx          *hcl.EvalContext
	requestBodyLimit int64
	reverseProxy     *httputil.ReverseProxy
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
	_ = p.setGetBody(req)

	var origin, hostname, path string
	httpContext := eval.NewHTTPContext(p.evalCtx, p.bufferOption, req, nil, nil)
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

// SetGetBody determines if we have to buffer a request body for further processing.
// First of all the user has a related reference within a config.Backend options declaration.
// Additionally the request body is nil or a NoBody type and the http method has no body restrictions like 'TRACE'.
func (p *Proxy) setGetBody(req *http.Request) error {
	if req.Method == http.MethodTrace {
		return nil
	}

	if (p.bufferOption & eval.BufferRequest) != eval.BufferRequest {
		return nil
	}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		buf := &bytes.Buffer{}
		lr := io.LimitReader(req.Body, p.requestBodyLimit+1)
		n, err := buf.ReadFrom(lr)
		if err != nil {
			return err
		}

		if n > p.requestBodyLimit {
			return errors.APIReqBodySizeExceeded
		}

		bodyBytes := buf.Bytes()
		req.GetBody = func() (io.ReadCloser, error) {
			return eval.NewReadCloser(bytes.NewBuffer(bodyBytes), req.Body), nil
		}
	}

	return nil
}

func getAttribute(ctx *hcl.EvalContext, name string, body *hcl.BodyContent) string {
	attr := body.Attributes
	if _, ok := attr[name]; !ok {
		return ""
	}
	originValue, _ := attr[name].Expr.Value(ctx)
	return seetie.ValueToString(originValue)
}
