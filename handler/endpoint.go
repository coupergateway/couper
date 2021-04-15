package handler

import (
	"context"
	"io"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
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
	var (
		clientres *http.Response
		err       error
		log       = e.log.WithField("uid", req.Context().Value(request.UID))
	)

	// Bind some values for logging purposes
	reqCtx := context.WithValue(req.Context(), request.Endpoint, e.opts.LogPattern)
	reqCtx = context.WithValue(reqCtx, request.EndpointKind, e.opts.LogHandlerKind)
	*req = *req.WithContext(reqCtx)

	defer func() {
		rc := recover()
		if rc != nil {
			log.WithField("panic", string(debug.Stack())).Error(rc)
			if clientres == nil {
				e.opts.Error.ServeError(errors.Server).ServeHTTP(rw, req)
			}
		}
	}()

	// subCtx is handled by this endpoint handler and should not be attached to req
	subCtx, cancel := context.WithCancel(reqCtx)
	defer cancel()

	if ee := eval.ApplyRequestContext(req.Context(), e.opts.Context, req); ee != nil {
		log.Error(ee)
	}

	proxyResults := make(producer.Results)
	requestResults := make(producer.Results)

	// go for it due to chan write on error
	go e.proxies.Produce(subCtx, req, proxyResults)
	go e.requests.Produce(subCtx, req, requestResults)

	beresps := make(producer.ResultMap)
	// TODO: read parallel, proxy first for now
	e.readResults(subCtx, proxyResults, beresps)
	e.readResults(subCtx, requestResults, beresps)

	evalContext := req.Context().Value(eval.ContextType).(*eval.Context)
	evalContext = evalContext.WithBeresps(beresps.List()...)

	// assume prio or err on conf load if set with response
	if e.redirect != nil {
		clientres = e.newRedirect()
	} else if e.response != nil {
		// TODO: refactor with error_handler, catch at least panics for now
		for _, b := range beresps {
			if b.Err == nil {
				continue
			}
			switch b.Err.(type) {
			case producer.ResultPanic:
				log.Error(b.Err)
			}
		}
		clientres, err = e.newResponse(req, evalContext)
	} else {
		if result, ok := beresps["default"]; ok {
			clientres = result.Beresp
			err = result.Err
		} else {
			// fallback
			err = errors.Configuration

			// TODO determine error priority, may solved with error_handler
			// on roundtrip panic the context label is missing atm
			// pick the first err from beresps
			for _, br := range beresps {
				if br != nil && br.Err != nil {
					err = br.Err
					break
				}
			}
		}
	}

	if err != nil {
		serveErr := err
		switch err.(type) { // TODO proper err mapping and handling
		case net.Error:
			serveErr = errors.Request.With(err)
			if p, ok := req.Context().Value(request.RoundTripProxy).(bool); ok && p {
				serveErr = errors.Proxy.With(err)
			}
		case producer.ResultPanic:
			serveErr = errors.Server.With(err)
			log.Error(err)
		}
		e.opts.Error.ServeError(serveErr).ServeHTTP(rw, req)
		return
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(evalContext, e.opts.Context, clientres); err != nil {
		log.Error(err)
	}

	select {
	case ctxErr := <-req.Context().Done():
		log.Errorf("endpoint write: %v", ctxErr)
	default:
	}

	if err = clientres.Write(rw); err != nil {
		log.Errorf("endpoint write: %v", err)
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

	log := e.log.WithField("uid", req.Context().Value(request.UID))

	hclCtx := evalCtx.HCLContext()

	content, _, diags := e.response.Context.PartialContent(config.ResponseInlineSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	statusCode := http.StatusOK
	if attr, ok := content.Attributes["status"]; ok {
		val, err := attr.Expr.Value(hclCtx)
		if err != nil {
			log.Error(errors.Evaluation.With(err).GoError())
			statusCode = http.StatusInternalServerError
		} else if statusValue := int(seetie.ValueToInt(val)); statusValue > 0 {
			statusCode = statusValue
		}
	}
	clientres.StatusCode = statusCode
	clientres.Status = http.StatusText(clientres.StatusCode)

	body, ct, bodyErr := eval.GetBody(hclCtx, content)
	if bodyErr != nil {
		log.Error(errors.Evaluation.With(bodyErr).GoError())
	}

	if ct != "" {
		clientres.Header.Set("Content-Type", ct)
	}

	if attr, ok := content.Attributes["headers"]; ok {
		val, err := attr.Expr.Value(hclCtx)
		if err != nil {
			log.Error(errors.Evaluation.With(err).GoError())
		}

		eval.SetHeader(val, clientres.Header)
	}

	if body != "" {
		r := strings.NewReader(body)
		clientres.Body = io.NopCloser(r)
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

func (e *Endpoint) readResults(ctx context.Context, requestResults producer.Results, beresps producer.ResultMap) {
	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		case r, more := <-requestResults:
			if !more {
				return
			}

			i++
			name := r.RoundTripName

			// fallback
			if name == "" { // panic case
				name = strconv.Itoa(i)
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
