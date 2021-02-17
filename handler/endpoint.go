package handler

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/avenga/couper/handler/producer"

	"github.com/hashicorp/hcl/v2"
)

type Endpoint struct {
	context  hcl.Body
	eval     *hcl.EvalContext
	proxy    []*producer.Proxy
	redirect *producer.Redirect
	requests *producer.Requests
	response *producer.Response
}

func NewEndpoint(requests []*producer.Request) *Endpoint {
	return &Endpoint{
		requests: producer.NewRequests(nil, requests, ""),
	}
}

func (e *Endpoint) ServerHTTP(rw http.ResponseWriter, req *http.Request) {
	subCtx, cancel := context.WithCancel(req.Context())
	defer cancel()

	// TODO: apply context (set/add/rm)
	//evalCtx := eval.NewHTTPContext(e.eval, eval.BufferNone, req, nil, nil)
	//e.context.JustAttributes()

	// A configured proxy is the only option within this endpoint, just serve.
	//if e.proxy != nil {
	//	if e.response == nil && e.redirect == nil {
	//		// apply context (set/add/rm) by proxy
	//		// TODO: how to with beresp endpoint eval
	//		e.proxy.ServeHTTP(rw, req)
	//		return
	//	}
	//}

	results := make(chan *producer.Result)
	//if e.proxy != nil {
	//	go func(r *http.Request, ch chan<- *result) {
	//		// TODO: replace with own rw, buffer | pipe opts
	//		rec := httptest.NewRecorder()
	//		// TODO: own result or map with result ch?
	//		e.proxy.ServeHTTP(rec, r)
	//		beresp := rec.Result() // TODO: missing roundtrip err
	//		ch <- &result{beresp: beresp}
	//	}(req, results)
	//}

	e.requests.Produce(subCtx, results)

	beresps := make(map[string]*producer.Result)
	var i int
	for r := range results { // collect resps
		if r == nil {
			panic("implement nil result handling")
		}
		// TODO: safe bereq access
		name, ok := r.Beresp.Request.Context().Value("requestName").(string)
		if !ok {
			name = "proxy"
		}
		beresps[strconv.Itoa(i)+name] = r
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
