package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/config/runtime/server"
	"github.com/coupergateway/couper/config/sequence"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/handler/producer"
	"github.com/coupergateway/couper/server/writer"
	"github.com/coupergateway/couper/telemetry"
	"github.com/coupergateway/couper/utils"
)

const serverTimingHeader = "Server-Timing"

var _ http.Handler = &Endpoint{}
var _ BodyLimit = &Endpoint{}

type Endpoint struct {
	log      *logrus.Entry
	modifier []hcl.Body
	opts     *EndpointOptions
}

type EndpointOptions struct {
	APIName           string
	BufferOpts        buffer.Option
	Context           *hclsyntax.Body
	ErrorTemplate     *errors.Template
	ErrorHandler      http.Handler
	IsErrorHandler    bool
	IsJob             bool
	LogHandlerKind    string
	LogPattern        string
	ReqBodyLimit      int64
	SendServerTimings bool
	ServerOpts        *server.Options

	Items     sequence.List
	Producers map[string]producer.Roundtrip
	Redirect  *producer.Redirect
	Response  *producer.Response
}

type BodyLimit interface {
	RequestLimit() int64
	BufferOptions() buffer.Option
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
	} else if e.opts.IsJob {
		clientres = &http.Response{StatusCode: http.StatusOK}
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

		w.AddModifier(e.modifier...)
		rw = w
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(httpCtx, e.opts.Context, clientres); err != nil {
		e.opts.ErrorTemplate.WithError(err).ServeHTTP(rw, req)
		return
	}

	if e.opts.SendServerTimings {
		st := getServerTimings(clientres.Header, beresps)
		if st != "" {
			rw.Header().Add(serverTimingHeader, st)
		}
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

func getServerTimings(headers http.Header, beresps producer.ResultMap) string {
	if len(beresps) == 0 {
		return ""
	}

	serverTimings := make(utils.ServerTimings)

	for _, h := range headers.Values(serverTimingHeader) {
		utils.CollectMetricNames(h, serverTimings)
	}

	for _, r := range beresps {
		if r == nil || r.Beresp == nil {
			continue
		}

		timings, _ := r.Beresp.Request.Context().Value(request.ServerTimings).(utils.ServerTimings)

		utils.MergeMetrics(timings, serverTimings)
	}

	var parts []string
	for k, v := range serverTimings {
		if v != "" {
			parts = append(parts, k+";"+v)
		}
	}

	return strings.Join(parts, ", ")
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

func newChannels(l sequence.List) (inputChannels, outputChannels map[string][]chan *producer.Result, resultChannels map[string]chan *producer.Result) {
	inputChannels = make(map[string][]chan *producer.Result)
	outputChannels = make(map[string][]chan *producer.Result)
	resultChannels = make(map[string]chan *producer.Result)
	for _, item := range l {
		fillChannels(item, inputChannels, outputChannels, resultChannels)
	}
	return inputChannels, outputChannels, resultChannels
}

func fillChannels(item *sequence.Item, inputChannels, outputChannels map[string][]chan *producer.Result, resultChannels map[string]chan *producer.Result) {
	for _, dep := range item.Deps() {
		ch := make(chan *producer.Result, 1)
		inputChannels[item.Name] = append(inputChannels[item.Name], ch)
		outputChannels[dep.Name] = append(outputChannels[dep.Name], ch)
		fillChannels(dep, inputChannels, outputChannels, resultChannels)
	}
	if _, ok := resultChannels[item.Name]; !ok {
		ch := make(chan *producer.Result, 1)
		outputChannels[item.Name] = append(outputChannels[item.Name], ch)
		resultChannels[item.Name] = ch
	}
}

func drainInputChannels(inputChannels, outputChannels []chan *producer.Result) bool {
	for _, inCh := range inputChannels {
		result := <-inCh
		if result.Err != nil {
			cErr, ok := result.Err.(*errors.Error)
			if !ok || !hasSequenceKind(cErr) {
				result.Err = errors.Sequence.With(cErr)
			}
			passToOutputChannels(result, outputChannels)
			return true
		}
	}
	return false
}

func hasSequenceKind(cerr *errors.Error) bool {
	for _, kind := range cerr.Kinds() {
		if kind == "sequence" {
			return true
		}
	}
	return false
}

func passToOutputChannels(result *producer.Result, outputChannels []chan *producer.Result) {
	for _, outCh := range outputChannels {
		outCh <- result
	}
}

type ResultPanic struct {
	err   error
	stack []byte
}

func (r ResultPanic) Error() string {
	return fmt.Sprintf("panic: %v\n%s", r.err, string(r.stack))
}

// produce hands over all possible outgoing requests to the producer interface and reads
// the backend response results afterward. Returns first occurred backend error.
func (e *Endpoint) produce(req *http.Request) (producer.ResultMap, error) {
	results := make(producer.ResultMap)

	outreq := req.WithContext(context.WithValue(req.Context(), request.ResponseBlock, e.opts.Response != nil))

	inputChannels, outputChannels, resultChannels := newChannels(e.opts.Items)
	sortedProducers := server.SortDefault(e.opts.Producers)
	for _, name := range sortedProducers {
		prod := e.opts.Producers[name]
		go func(n string, rt producer.Roundtrip, intChs, outChs []chan *producer.Result) {
			defer func() {
				if rp := recover(); rp != nil {
					res := &producer.Result{
						Err: ResultPanic{
							err:   fmt.Errorf("%v", rp),
							stack: debug.Stack(),
						},
						RoundTripName: n,
					}
					passToOutputChannels(res, outChs)
				}
			}()

			if drainInputChannels(intChs, outChs) {
				return
			}
			res := rt.Produce(outreq)
			passToOutputChannels(res, outChs)
		}(name, prod, inputChannels[name], outputChannels[name])
		time.Sleep(time.Millisecond * 2)
	}
	readResults(e.opts.Items, resultChannels, results)

	var err error // TODO: prefer default resp err
	// TODO: additionally log all panic error types
	for _, r := range results {
		if r != nil && r.Err != nil {
			err = r.Err
			break
		}
	}

	return results, err
}

func readResults(items sequence.List, resultChannels map[string]chan *producer.Result, beresps producer.ResultMap) {
	for _, item := range items {
		res := <-resultChannels[item.Name]
		beresps[item.Name] = res
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
	case ResultPanic:
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

	// modify response status code if set
	if attr, ok := e.opts.Context.Attributes["set_response_status"]; e.opts.IsErrorHandler && ctxErr == err && ok {
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

func (e *Endpoint) BufferOptions() buffer.Option {
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
