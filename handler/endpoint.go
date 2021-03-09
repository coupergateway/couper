package handler

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/internal/seetie"
)

var _ http.Handler = &Endpoint{}
var _ EndpointLimit = &Endpoint{}

type Endpoint struct {
	log            *logrus.Entry
	logHandlerKind string
	opts           *EndpointOptions
	proxies        producer.Roundtrips
	redirect       *producer.Redirect
	requests       producer.Roundtrips
	response       *producer.Response
}

type EndpointOptions struct {
	Context        hcl.Body
	Error          *errors.Template
	LogHandlerKind string
	LogPattern     string
	ReqBodyLimit   int64
	ReqBufferOpts  eval.BufferOption
	ServerOpts     *server.Options
}

type EndpointLimit interface {
	RequestLimit() int64
}

func NewEndpoint(opts *EndpointOptions, log *logrus.Entry, proxies producer.Proxies,
	requests producer.Requests, resp *producer.Response) *Endpoint {
	opts.ReqBufferOpts |= eval.MustBuffer(opts.Context) // TODO: proper configuration on all hcl levels
	return &Endpoint{
		log:      log.WithField("handler", opts.LogHandlerKind),
		opts:     opts,
		proxies:  proxies,
		requests: requests,
		response: resp,
	}
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Bind some values for logging purposes
	reqCtx := context.WithValue(req.Context(), request.Endpoint, e.opts.LogPattern)
	reqCtx = context.WithValue(req.Context(), request.EndpointKind, e.opts.LogHandlerKind)
	*req = *req.WithContext(reqCtx)

	// subCtx is handled by this endpoint handler and should not be attached to req
	subCtx, cancel := context.WithCancel(reqCtx)
	defer cancel()

	if ee := eval.ApplyRequestContext(req.Context(), e.opts.Context, req); ee != nil {
		e.log.Error(ee)
	}

	proxyResults := make(producer.Results)
	requestResults := make(producer.Results)

	// go for it due to chan write on error
	go e.proxies.Produce(subCtx, req, proxyResults)
	go e.requests.Produce(subCtx, req, requestResults)

	beresps := make(producer.ResultMap)
	// TODO: read parallel, proxy first for now
	e.readResults(proxyResults, beresps)
	e.readResults(requestResults, beresps)

	var clientres *http.Response
	var err error

	evalContext := req.Context().Value(eval.ContextType).(*eval.Context)
	evalContext = evalContext.WithBeresps(beresps.List()...)

	// assume prio or err on conf load if set with response
	if e.redirect != nil {
		clientres = e.newRedirect()
	} else if e.response != nil {
		clientres, err = e.newResponse(req, evalContext)
	} else {
		if result, ok := beresps["default"]; ok {
			clientres = result.Beresp
			err = result.Err
		} else {
			err = errors.Configuration
		}
	}

	if err != nil {
		serveErr := err
		switch err.(type) { // TODO proper err mapping and handling
		case net.Error:
			serveErr = errors.EndpointConnect
			if p, ok := req.Context().Value(request.RoundTripProxy).(bool); ok && p {
				serveErr = errors.EndpointProxyConnect
			}
		}
		e.opts.Error.ServeError(serveErr).ServeHTTP(rw, req)
		return
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(evalContext, e.opts.Context, clientres); err != nil {
		e.log.Error(err)
	}

	if err = clientres.Write(rw); err != nil {
		e.log.Errorf("endpoint write error: %v", err)
	}
}

func (e *Endpoint) newResponse(req *http.Request, evalCtx *eval.Context) (*http.Response, error) {
	clientres := &http.Response{
		Header:     make(http.Header),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
	}

	hclCtx := evalCtx.HCLContext()

	content, _, diags := e.response.Context.PartialContent(config.ResponseInlineSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	statusCode := http.StatusOK
	if attr, ok := content.Attributes["status"]; ok {
		val, _ := attr.Expr.Value(hclCtx)
		statusCode = int(seetie.ValueToInt(val))
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	if attr, ok := content.Attributes["headers"]; ok {
		val, _ := attr.Expr.Value(hclCtx)
		eval.SetHeader(val, clientres.Header)
	}

	if attr, ok := content.Attributes["body"]; ok {
		val, _ := attr.Expr.Value(hclCtx)
		r := strings.NewReader(seetie.ValueToString(val))
		clientres.Body = eval.NewReadCloser(r, nil)
	}

	return clientres, nil
}

func (e *Endpoint) newRedirect() *http.Response {
	// TODO use http.RedirectHandler
	status := http.StatusMovedPermanently
	return &http.Response{
		//Header: e.redirect.Header,
		//Body:   e.redirect.Body, // TODO: closeWrapper
		StatusCode: status,
	}
}

func (e *Endpoint) readResults(requestResults producer.Results, beresps producer.ResultMap) {
	for r := range requestResults { // collect resps
		if r == nil {
			panic("implement nil result handling")
		}

		if r.Beresp != nil {
			ctx := r.Beresp.Request.Context()
			var name string
			if n, ok := ctx.Value(request.RoundTripName).(string); ok && n != "" {
				name = n
			}
			// fallback
			if name == "" {
				if id, ok := ctx.Value(request.UID).(string); ok {
					name = id
				}
			}
			beresps[name] = r
		}
	}
}

func (e *Endpoint) Options() *server.Options {
	return e.opts.ServerOpts
}

func (e *Endpoint) RequestLimit() int64 {
	return e.opts.ReqBodyLimit
}

// String interface maps to the access log handler field.
func (e *Endpoint) String() string {
	return e.logHandlerKind
}
