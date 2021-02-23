package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
)

var _ http.Handler = &Endpoint{}

type Endpoint struct {
	context  hcl.Body
	eval     *hcl.EvalContext
	log      *logrus.Entry
	proxies  producer.Roundtrips
	redirect *producer.Redirect
	requests producer.Roundtrips
	response *producer.Response
}

func NewEndpoint(ctx hcl.Body, evalCtx *hcl.EvalContext, log *logrus.Entry,
	proxies producer.Proxies, requests producer.Requests) *Endpoint {
	return &Endpoint{
		context:  ctx,
		eval:     evalCtx,
		log:      log,
		proxies:  proxies,
		requests: requests,
	}
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	subCtx, cancel := context.WithCancel(req.Context())
	defer cancel()

	if ee := eval.ApplyRequestContext(e.eval, e.context, req); ee != nil {
		e.log.Error(ee)
	}

	proxyResults := make(producer.Results)
	requestResults := make(producer.Results)

	// go for it due to chan write on error
	go e.proxies.Produce(subCtx, req, e.eval, proxyResults)
	go e.requests.Produce(subCtx, req, e.eval, requestResults)

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
		clientres = e.newResponse(beresps)
	} else {
		if len(beresps) > 1 {
			e.log.Error("endpoint configuration error")
			return
		}
		for _, result := range beresps {
			clientres = result.Beresp
			err = result.Err
			break
		}
	}

	if err != nil {
		e.log.Errorf("upstream error: %v", err)
		return
	}

	// always apply before write: redirect, response
	if err = eval.ApplyResponseContext(e.eval, e.context, req, clientres); err != nil {
		e.log.Error(err)
	}

	if err = clientres.Write(rw); err != nil {
		e.log.Errorf("endpoint write error: %v", err)
	}
}

func (e *Endpoint) newResponse(beresps map[string]*producer.Result) *http.Response {
	// TODO: beresps.eval....
	clientres := &http.Response{
		StatusCode: e.response.Status,
		Header:     e.response.Header,
	}
	return clientres
}

func (e *Endpoint) newRedirect() *http.Response {
	// TODO use http.RedirectHandler
	status := http.StatusMovedPermanently
	if e.redirect.Status > 0 {
		status = e.redirect.Status
	}
	return &http.Response{
		Header: e.redirect.Header,
		//Body:   e.redirect.Body, // TODO: closeWrapper
		StatusCode: status,
	}
}

func (e *Endpoint) readResults(requestResults producer.Results, beresps map[string]*producer.Result) {
	i := 0
	for r := range requestResults { // collect resps
		if r == nil {
			panic("implement nil result handling")
		}
		// TODO: safe bereq access
		name, ok := r.Beresp.Request.Context().Value("requestName").(string)
		if !ok {
			name = "proxy"
		}
		beresps[strconv.Itoa(i)+name] = r
		i++
	}
}
