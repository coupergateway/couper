package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	context  hcl.Body
	eval     *hcl.EvalContext
	proxy    *Proxy // TODO: proxy with backend struct
	redirect *Redirect
	requests []*Request
	response *Response
}

type result struct {
	beresp *http.Response
	err    error
	// TODO: trace
}

type endpointResults chan *result

func (e *Endpoint) ServerHTTP(rw http.ResponseWriter, req *http.Request) {
	subCtx, cancel := context.WithCancel(req.Context())
	defer cancel()

	// TODO: apply context (set/add/rm)
	//evalCtx := eval.NewHTTPContext(e.eval, eval.BufferNone, req, nil, nil)
	//e.context.JustAttributes()

	// A configured proxy is the only option within this endpoint, just serve.
	if e.proxy != nil {
		if e.response == nil && e.redirect == nil {
			// apply context (set/add/rm) by proxy
			// TODO: how to with beresp endpoint eval
			e.proxy.ServeHTTP(rw, req)
			return
		}
	}

	requests := e.requests[:]
	results := make(endpointResults, len(requests))

	if e.proxy != nil {
		go func(r *http.Request, ch chan<- *result) {
			// TODO: replace with own rw, buffer | pipe opts
			rec := httptest.NewRecorder()
			// TODO: own result or map with result ch?
			e.proxy.ServeHTTP(rec, r)
			beresp := rec.Result() // TODO: missing roundtrip err
			ch <- &result{beresp: beresp}
		}(req, results)
	}

	for _, or := range requests {
		go e.roundtrip(subCtx, or, results)
	}

	beresps := make(map[string]*result)
	for r := range results { // collect resps
		if r == nil {
			panic("implement nil result handling")
		}
		// TODO: safe bereq access
		name, ok := r.beresp.Request.Context().Value("requestName").(string)
		if !ok {
			name = "proxy"
		}
		beresps[name] = r
	}

	// TODO: apply context with beresps
	//res := &http.Response{
	//	Proto: req.Proto,
	//}
	//res.Write(rw)

	// assume prio or err on conf load if set with response
	//if e.redirect != nil {
	//	e.redirect.Write(rw)
	//	return
	//}

	if e.response != nil {
		for k, v := range e.response.Header {
			rw.Header()[k] = v
		}
		rw.WriteHeader(e.response.Status)
		io.Copy(rw, e.response.Body) // TODO: err handling
	}
}

func (e *Endpoint) roundtrip(ctx context.Context, req *Request, results chan<- *result) {
	outreq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		results <- &result{err: err}
		return
	}

	// TODO: apply context to outreq

	deadlineCtx, cancelDeadLine := context.WithTimeout(ctx, time.Second*10)
	defer cancelDeadLine()

	deadlineCtx = context.WithValue(deadlineCtx, "requestName", req.Name) // TODO: key const

	beresp, err := req.Backend.RoundTrip(outreq.WithContext(deadlineCtx)) // backend hop, oauth, openapi etc.
	r := &result{
		beresp: beresp,
		err:    err,
	}

	// TODO: apply context to beresp

	results <- r
}
