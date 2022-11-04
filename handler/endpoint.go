package handler

import (
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/server/writer"
	"github.com/avenga/couper/telemetry"
)

var _ http.Handler = &Endpoint{}
var _ BodyLimit = &Endpoint{}

type Endpoint struct {
	log      *logrus.Entry
	modifier []hcl.Body
	opts     *EndpointOptions
}

type EndpointOptions struct {
	APIName        string
	BufferOpts     eval.BufferOption
	Context        hcl.Body
	ErrorTemplate  *errors.Template
	ErrorHandler   http.Handler
	IsErrorHandler bool
	LogHandlerKind string
	LogPattern     string
	ReqBodyLimit   int64
	ServerOpts     *server.Options

	Proxies   producer.Roundtrip
	Redirect  *producer.Redirect
	Requests  producer.Roundtrip
	Sequences producer.Sequences
	Response  *producer.Response
}

type BodyLimit interface {
	RequestLimit() int64
	BufferOptions() eval.BufferOption
}

func NewEndpoint(opts *EndpointOptions, log *logrus.Entry, modifier []hcl.Body) *Endpoint {
	return &Endpoint{
		log:      log.WithField("handler", opts.LogHandlerKind),
		modifier: modifier,
		opts:     opts,
	}
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqCtx := e.withContext(req)

	var (
		clientres *http.Response
		err       error
		log       = e.log.WithContext(reqCtx)
	)

	defer func() {
		if rc := recover(); rc != nil {
			log.WithField("panic", string(debug.Stack())).Error(rc)
			if clientres == nil {
				e.opts.ErrorTemplate.WithError(errors.Server).ServeHTTP(rw, req)
			}
		}
	}()

	if e.opts.LogPattern != "" {
		span := trace.SpanFromContext(reqCtx)
		span.SetAttributes(telemetry.KeyEndpoint.String(e.opts.LogPattern))
	}

	if ee := eval.ApplyRequestContext(eval.ContextFromRequest(req).HCLContext(), e.opts.Context, req); ee != nil {
		e.opts.ErrorTemplate.WithError(ee).ServeHTTP(rw, req)
		return
	}

	// subCtx is handled by this endpoint handler and should not be attached to req
	subCtx, cancel := context.WithCancel(reqCtx)
	defer cancel()

	beresps, err := e.produce(req.WithContext(subCtx))

	// check for client cancels before reading backend response bodies
	select {
	case <-reqCtx.Done():
		err = reqCtx.Err()
		log.WithError(errors.ClientRequest.With(err)).Error()
		return
	default:
	}

	// handle errors first before entering the happy path
	if !e.opts.IsErrorHandler {
		if handled := e.handleError(rw, req, err); handled {
			return
		}
	}

	// assume configured priority, prefer redirect to response and default ones
	if e.opts.Redirect != nil {
		clientres = e.newRedirect()
	} else if e.opts.Response != nil {
		_, span := telemetry.NewSpanFromContext(subCtx, "response", trace.WithSpanKind(trace.SpanKindProducer))
		defer span.End()
		clientres, err = producer.NewResponse(req, e.opts.Response.Context, http.StatusOK)
	} else if result, exist := beresps["default"]; exist {
		clientres = result.Beresp
		err = result.Err
	} else if e.opts.IsErrorHandler && err == nil {
		var ok bool
		err, ok = req.Context().Value(request.Error).(error)
		if !ok {
			err = errors.Server
		}
	} else {
		err = errors.Server.Message("missing client response")
	}

	if handled := e.handleError(rw, req, err); handled {
		return
	}

	select {
	case ctxErr := <-req.Context().Done():
		log.Errorf("endpoint write: %v", ctxErr)
	default:
	}

	httpCtx := eval.ContextFromRequest(req).HCLContextSync()

	w, ok := rw.(*writer.Response)
	if !ok {
		log.Errorf("response writer: type error")
	} else {
		// 'clientres' is a faulty response object due to a websocket hijack.
		if w.IsHijacked() {
			return
		}

		w.AddModifier(httpCtx, e.modifier...)
		rw = w
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(httpCtx, e.opts.Context, clientres); err != nil {
		e.opts.ErrorTemplate.WithError(err).ServeHTTP(rw, req)
		return
	}

	// copy/write like a reverseProxy
	copyHeader(rw.Header(), clientres.Header)

	rw.WriteHeader(clientres.StatusCode)

	if clientres.Body == nil {
		return
	}

	err = copyResponse(rw, clientres.Body, flushInterval(clientres))
	if err != nil {
		// Since we're streaming the response, if we run into an error all we can do
		// is abort the request.
		log.WithError(errors.Server.With(err).Message("body copy failed")).Error()
	}

	_ = clientres.Body.Close()
}

// withContext binds some endpoint context related values for logging and buffer purposes.
func (e *Endpoint) withContext(req *http.Request) context.Context {
	reqCtx := context.WithValue(req.Context(), request.Endpoint, e.opts.LogPattern)
	reqCtx = context.WithValue(reqCtx, request.EndpointKind, e.opts.LogHandlerKind)
	reqCtx = context.WithValue(reqCtx, request.APIName, e.opts.APIName)
	reqCtx = context.WithValue(reqCtx, request.BufferOptions, e.opts.BufferOpts)
	*req = *req.WithContext(reqCtx)
	return reqCtx
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

// produce hands over all possible outgoing requests to the producer interface and reads
// the backend response results afterwards. Returns first occurred backend error.
func (e *Endpoint) produce(req *http.Request) (producer.ResultMap, error) {
	results := make(producer.ResultMap)

	outreq := req.WithContext(context.WithValue(req.Context(), request.ResponseBlock, e.opts.Response != nil))

	trips := []producer.Roundtrip{e.opts.Proxies, e.opts.Requests, e.opts.Sequences}
	tripCh := make(chan chan *producer.Result, len(trips))
	for _, trip := range trips {
		// use-case: just a response block within an endpoint
		if trip == nil {
			continue
		}

		resultCh := make(chan *producer.Result, trip.Len())
		go func(rt producer.Roundtrip, rc chan *producer.Result) {
			rt.Produce(outreq, rc)
			close(rc)
		}(trip, resultCh)
		tripCh <- resultCh
	}
	close(tripCh)

	for resultCh := range tripCh {
		e.readResults(req.Context(), resultCh, results)
	}

	var err error // TODO: prefer default resp err
	// TODO: additionally log all panic error types
	for _, r := range results {
		if r.Err != nil {
			err = r.Err
			break
		}
	}

	return results, err
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

func (e *Endpoint) handleError(rw http.ResponseWriter, req *http.Request, err error) bool {
	if err == nil {
		return false
	}

	ctxErr := req.Context().Value(request.Error)
	serveErr := err
	switch err.(type) {
	case net.Error:
		serveErr = errors.Request.With(err)
		if p, ok := req.Context().Value(request.RoundTripProxy).(bool); ok && p {
			serveErr = errors.Proxy.With(err)
		}
	case producer.ResultPanic:
		serveErr = errors.Server.With(err)
	}

	if e.opts.ErrorHandler != nil {
		if ctxErr == nil {
			ctxErr = serveErr
			*req = *req.WithContext(context.WithValue(req.Context(), request.Error, ctxErr))
		}
		e.opts.ErrorHandler.ServeHTTP(rw, req)
		return true
	}

	content, _, _ := e.opts.Context.PartialContent(config.Endpoint{}.Schema(true))

	// modify response status code if set
	if attr, ok := content.Attributes["set_response_status"]; e.opts.IsErrorHandler && ctxErr == err && ok {
		if statusCode, applyErr := eval.
			ApplyResponseStatus(eval.ContextFromRequest(req).HCLContextSync(), attr, nil); statusCode > 0 {
			if serr, k := serveErr.(*errors.Error); k {
				serveErr = serr.Status(statusCode)
			} else {
				serveErr = errors.Server.With(serveErr).Status(statusCode)
			}
		} else if applyErr != nil {
			e.log.WithError(applyErr)
		}
	}

	e.opts.ErrorTemplate.WithError(serveErr).ServeHTTP(rw, req)
	return true
}

func (e *Endpoint) Options() *server.Options {
	return e.opts.ServerOpts
}

func (e *Endpoint) BufferOptions() eval.BufferOption {
	return e.opts.BufferOpts
}

// BodyContext exposes the current endpoint hcl.Body.
func (e *Endpoint) BodyContext() hcl.Body {
	return e.opts.Context
}

func (e *Endpoint) RequestLimit() int64 {
	return e.opts.ReqBodyLimit
}

// String interface maps to the access log handler field.
func (e *Endpoint) String() string {
	return e.opts.LogHandlerKind
}
