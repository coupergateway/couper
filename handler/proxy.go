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
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

// Proxy represents the producer <Proxy> object.
type Proxy struct {
	Backend          *transport.Backend
	bufferOption     eval.BufferOption
	requestBodyLimit int64
	Context          hcl.Body
	evalCtx          *hcl.EvalContext
	Proxy            *httputil.ReverseProxy
}

func NewProxy(backend *transport.Backend, ctx hcl.Body, evalCtx *hcl.EvalContext) *Proxy {
	proxy := &Proxy{
		Backend: backend,
		Context: ctx,
		evalCtx: evalCtx,
	}
	p := &httputil.ReverseProxy{
		Director:       proxy.director,
		ModifyResponse: proxy.modifyResponse,
		Transport:      proxy,
	}
	proxy.Proxy = p
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	return p.Backend.RoundTrip(req)
}

var backendInlineSchema = config.Backend{}.Schema(true)

func (p *Proxy) director(req *http.Request) {
	_ = p.setGetBody(req)

	var origin, hostname, path string
	evalContext := eval.NewHTTPContext(p.evalCtx, p.bufferOption, req, nil, nil)

	content, _, _ := p.Context.PartialContent(backendInlineSchema)
	if o := getAttribute(evalContext, "origin", content); o != "" {
		origin = o
	}
	if h := getAttribute(evalContext, "hostname", content); h != "" {
		hostname = h
	}
	if pathVal := getAttribute(evalContext, "path", content); pathVal != "" {
		path = pathVal
	}

	//if origin == "" {
	//	return errors.New("proxy: origin not set")
	//}

	originURL, _ := url.Parse(origin)
	//if err != nil {
	//	return fmt.Errorf("proxy: parse origin: %w", err)
	//}

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

func (p *Proxy) modifyResponse(res *http.Response) error {
	return nil
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
			//return p.getErrorCode(couperErr.APIReqBodySizeExceeded)
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
