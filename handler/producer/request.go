package producer

import (
	"context"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

// Request represents the producer <Request> object.
type Request struct {
	Backend   http.RoundTripper
	Context   *hclsyntax.Body
	Name      string // label
	dependsOn string
}

func (r *Request) Produce(req *http.Request) *Result {
	hclCtx := eval.ContextFromRequest(req).HCLContextSync() // also synced for requests due to sequence case

	outCtx := withRoundTripName(req.Context(), r.Name)
	if r.dependsOn != "" {
		outCtx = context.WithValue(outCtx, request.EndpointSequenceDependsOn, r.dependsOn)
	}

	methodVal, err := eval.ValueFromBodyAttribute(hclCtx, r.Context, "method")
	if err != nil {
		return &Result{Err: err}
	}
	method := seetie.ValueToString(methodVal)

	outreq := req.Clone(req.Context())
	removeHost(outreq)

	url, err := NewURLFromAttribute(hclCtx, r.Context, "url", outreq)
	if err != nil {
		return &Result{Err: err}
	}

	body, defaultContentType, err := eval.GetBody(hclCtx, r.Context)
	if err != nil {
		return &Result{Err: err}
	}

	if method == "" {
		method = http.MethodGet

		if len(body) > 0 {
			method = http.MethodPost
		}
	}

	outreq, err = http.NewRequest(strings.ToUpper(method), url.String(), nil)
	if err != nil {
		return &Result{Err: err}
	}

	expStatusVal, err := eval.ValueFromBodyAttribute(hclCtx, r.Context, "expected_status")
	if err != nil {
		return &Result{Err: err}
	}

	outCtx = context.WithValue(outCtx, request.EndpointExpectedStatus, seetie.ValueToIntSlice(expStatusVal))

	if defaultContentType != "" {
		outreq.Header.Set("Content-Type", defaultContentType)
	}

	eval.SetBody(outreq, []byte(body))

	outreq = outreq.WithContext(outCtx)
	err = eval.ApplyRequestContext(hclCtx, r.Context, outreq)
	if err != nil {
		return &Result{Err: err}
	}

	return roundtrip(r.Backend, outreq)
}

func (r *Request) SetDependsOn(ps string) {
	r.dependsOn = ps
}

func withRoundTripName(ctx context.Context, name string) context.Context {
	n := name
	if n == "" {
		n = "default"
	}
	return context.WithValue(ctx, request.RoundTripName, n)
}
