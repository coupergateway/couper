package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/internal/seetie"
)

var _ http.Handler = &Endpoint{}

const defaultReqBodyLimit = "64MiB"

type Endpoint struct {
	evalContext    *hcl.EvalContext
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
}

func NewEndpoint(opts *EndpointOptions, evalCtx *hcl.EvalContext, log *logrus.Entry,
	proxies producer.Proxies, requests producer.Requests, resp *producer.Response) *Endpoint {
	opts.ReqBufferOpts |= eval.MustBuffer(opts.Context) // TODO: proper configuration on all hcl levels
	return &Endpoint{
		evalContext: evalCtx,
		log:         log.WithField("handler", opts.LogHandlerKind),
		opts:        opts,
		proxies:     proxies,
		requests:    requests,
		response:    resp,
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

	if err := e.SetGetBody(req); err != nil {
		e.opts.Error.ServeError(err).ServeHTTP(rw, req)
		return
	}

	if ee := eval.ApplyRequestContext(e.evalContext, e.opts.Context, req); ee != nil {
		e.log.Error(ee)
	}

	proxyResults := make(producer.Results)
	requestResults := make(producer.Results)

	// go for it due to chan write on error
	go e.proxies.Produce(subCtx, req, e.evalContext, proxyResults)
	go e.requests.Produce(subCtx, req, e.evalContext, requestResults)

	beresps := make(map[string]*producer.Result)
	// TODO: read parallel, proxy first for now
	e.readResults(proxyResults, beresps)
	e.readResults(requestResults, beresps)

	var clientres *http.Response
	var err error

	// assume prio or err on conf load if set with response
	if e.redirect != nil {
		clientres = e.newRedirect()
	} else if e.response != nil {
		clientres, err = e.newResponse(req, beresps)
	} else {
		if result, ok := beresps["default"]; ok {
			clientres = result.Beresp
			err = result.Err
		} else {
			err = errors.Configuration
		}
	}

	if err != nil {
		e.opts.Error.ServeError(err).ServeHTTP(rw, req)
		return
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(e.evalContext, e.opts.Context, req, clientres); err != nil {
		e.log.Error(err)
	}

	if err = clientres.Write(rw); err != nil {
		e.log.Errorf("endpoint write error: %v", err)
	}
}

// SetGetBody determines if we have to buffer a request body for further processing.
// First of all the user has a related reference within a related options context declaration.
// Additionally the request body is nil or a NoBody type and the http method has no body restrictions like 'TRACE'.
func (e *Endpoint) SetGetBody(req *http.Request) error {
	if req.Method == http.MethodTrace {
		return nil
	}

	// TODO: handle buffer options based on overall body context and reference
	//if (e.opts.ReqBufferOpts & eval.BufferRequest) != eval.BufferRequest {
	//	return nil
	//}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		buf := &bytes.Buffer{}
		lr := io.LimitReader(req.Body, e.opts.ReqBodyLimit+1)
		n, err := buf.ReadFrom(lr)
		if err != nil {
			return err
		}

		if n > e.opts.ReqBodyLimit {
			return errors.APIReqBodySizeExceeded
		}

		bodyBytes := buf.Bytes()
		req.GetBody = func() (io.ReadCloser, error) {
			return eval.NewReadCloser(bytes.NewBuffer(bodyBytes), req.Body), nil
		}
	}

	return nil
}

func (e *Endpoint) newResponse(req *http.Request, beresps map[string]*producer.Result) (*http.Response, error) {
	var resps []*http.Response
	for _, br := range beresps {
		resps = append(resps, br.Beresp)
	}

	clientres := &http.Response{
		Header:     make(http.Header),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
	}

	httpCtx := eval.NewHTTPContext(e.evalContext, e.opts.ReqBufferOpts, req, resps...)
	content, _, diags := e.response.Context.PartialContent(config.ResponseInlineSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	statusCode := http.StatusOK
	if attr, ok := content.Attributes["status"]; ok {
		val, _ := attr.Expr.Value(httpCtx)
		statusCode = int(seetie.ValueToInt(val))
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	if attr, ok := content.Attributes["headers"]; ok {
		val, _ := attr.Expr.Value(httpCtx)
		eval.SetHeader(val, clientres.Header)
	}

	if attr, ok := content.Attributes["body"]; ok {
		val, _ := attr.Expr.Value(httpCtx)
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

func (e *Endpoint) readResults(requestResults producer.Results, beresps map[string]*producer.Result) {
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

// String interface maps to the access log handler field.
func (e *Endpoint) String() string {
	return e.logHandlerKind
}

func ParseBodyLimit(limit string) (int64, error) {
	requestBodyLimit := defaultReqBodyLimit
	if limit != "" {
		requestBodyLimit = limit
	}
	return units.FromHumanSize(requestBodyLimit)
}
