package handler

import (
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

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
var _ EndpointLimit = &Endpoint{}

type Endpoint struct {
	log      *logrus.Entry
	modifier []hcl.Body
	opts     *EndpointOptions
}

type EndpointOptions struct {
	APIName        string
	Bodies         []hcl.Body
	Context        hcl.Body
	Error          *errors.Template
	LogHandlerKind string
	LogPattern     string
	ReqBodyLimit   int64
	ReqBufferOpts  eval.BufferOption
	ServerOpts     *server.Options

	Proxies  producer.Roundtrips
	Redirect *producer.Redirect
	Requests producer.Roundtrips
	Response *producer.Response
}

type EndpointLimit interface {
	RequestLimit() int64
}

func NewEndpoint(opts *EndpointOptions, log *logrus.Entry, modifier []hcl.Body) *Endpoint {
	opts.ReqBufferOpts |= eval.MustBuffer(opts.Context) // TODO: proper configuration on all hcl levels
	return &Endpoint{
		log:      log.WithField("handler", opts.LogHandlerKind),
		modifier: modifier,
		opts:     opts,
	}
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var (
		clientres    *http.Response
		err          error
		log          = e.log.WithContext(req.Context())
		isErrHandler = strings.HasPrefix(e.opts.LogHandlerKind, "error_") // weak ref
	)

	// Bind some values for logging purposes
	reqCtx := context.WithValue(req.Context(), request.Endpoint, e.opts.LogPattern)
	reqCtx = context.WithValue(reqCtx, request.EndpointKind, e.opts.LogHandlerKind)
	reqCtx = context.WithValue(reqCtx, request.APIName, e.opts.APIName)

	*req = *req.WithContext(reqCtx)
	if e.opts.LogPattern != "" {
		span := trace.SpanFromContext(reqCtx)
		span.SetAttributes(telemetry.KeyEndpoint.String(e.opts.LogPattern))
	}

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

	if ee := eval.ApplyRequestContext(reqCtx, e.opts.Context, req); ee != nil {
		e.opts.Error.ServeError(ee).ServeHTTP(rw, req)
		return
	}

	proxyResults := make(producer.Results)
	requestResults := make(producer.Results)

	// go for it due to chan write on error
	go e.opts.Proxies.Produce(subCtx, req, proxyResults)
	go e.opts.Requests.Produce(subCtx, req, requestResults)

	beresps := make(producer.ResultMap)
	// TODO: read parallel, proxy first for now
	e.readResults(subCtx, proxyResults, beresps)
	e.readResults(subCtx, requestResults, beresps)

	select {
	case <-reqCtx.Done():
		err = reqCtx.Err()
		log.WithError(errors.ClientRequest.With(err)).Error()
		return
	default:
	}

	evalContext := eval.ContextFromRequest(req)
	evalContext = evalContext.WithBeresps(beresps.List()...)

	// assume prio or err on conf load if set with response
	if e.opts.Redirect != nil {
		clientres = e.newRedirect()
	} else if e.opts.Response != nil {
		// TODO: refactor with error_handler, catch at least panics for now
		for _, b := range beresps {
			if b.Err == nil {
				continue
			}

			switch b.Err.(type) {
			case producer.ResultPanic:
				log.WithError(b.Err).Error()
			}

			if b.Err != nil {
				err = b.Err
				break
			}
		}

		if err == nil {
			_, span := telemetry.NewSpanFromContext(subCtx, "response", trace.WithSpanKind(trace.SpanKindProducer))
			defer span.End()
			clientres, err = producer.NewResponse(req, e.opts.Response.Context, evalContext, http.StatusOK)
		}
	} else {
		if result, ok := beresps["default"]; ok {
			clientres = result.Beresp
			err = result.Err
		} else {
			// fallback
			err = errors.Configuration

			if isErrHandler {
				err = req.Context().Value(request.Error).(*errors.Error)
			} else {
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
			log.WithError(err).Error()
		}

		content, _, _ := e.opts.Context.PartialContent(config.Endpoint{}.Schema(true))
		errFromCtx := req.Context().Value(request.Error)
		if attr, ok := content.Attributes["set_response_status"]; isErrHandler && errFromCtx == err && ok {
			if statusCode, applyErr := eval.ApplyResponseStatus(evalContext, attr, nil); statusCode > 0 {
				if serr, k := serveErr.(*errors.Error); k {
					serveErr = serr.Status(statusCode)
				} else {
					serveErr = errors.Server.With(serveErr).Status(statusCode)
				}
			} else if applyErr != nil {
				e.log.WithError(applyErr)
			}
		}

		e.opts.Error.ServeError(serveErr).ServeHTTP(rw, req)
		return
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(evalContext, e.opts.Context, clientres); err != nil {
		e.opts.Error.ServeError(err).ServeHTTP(rw, req)
		return
	}

	eval.ApplyCustomLogs(evalContext, e.opts.Bodies, req, e.log, request.AccessLogFields)

	select {
	case ctxErr := <-req.Context().Done():
		log.Errorf("endpoint write: %v", ctxErr)
	default:
	}

	w, ok := rw.(*writer.Response)
	if !ok {
		log.Errorf("response writer: type error")
	} else {
		if w.IsHijacked() {
			// clientres is a faulty response object due to a websocket hijack.
			return
		}

		w.AddModifier(evalContext, e.modifier)
		rw = w
	}

	if err = clientres.Write(rw); err != nil {
		log.Errorf("endpoint write: %v", err)
	}
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
	return e.opts.LogHandlerKind
}
