package producer

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
)

type Proxy struct {
	Content   *hclsyntax.Body
	Name      string // label
	dependsOn string
	RoundTrip http.RoundTripper
}

func (p *Proxy) Produce(clientReq *http.Request) *Result {
	outCtx := withRoundTripName(clientReq.Context(), p.Name)
	outCtx = context.WithValue(outCtx, request.RoundTripProxy, true)
	if p.dependsOn != "" {
		outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, p.dependsOn)
	}

	// since proxy and backend may work on the "same" outReq this must be cloned.
	outReq := clientReq.Clone(outCtx)
	removeHost(outReq)

	hclCtx := eval.ContextFromRequest(clientReq).HCLContext()
	url, err := NewURLFromAttribute(hclCtx, p.Content, "url", outReq)
	if err != nil {
		return &Result{Err: err}
	}

	// proxy should pass query if not redefined with url attribute
	if outReq.URL.RawQuery != "" && url.RawQuery == "" {
		url.RawQuery = outReq.URL.RawQuery
	}

	outReq.URL = url

	return roundtrip(p.RoundTrip, outReq)
}

func (p *Proxy) SetDependsOn(ps string) {
	p.dependsOn = ps
}
