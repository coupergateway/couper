package transport

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/hashicorp/hcl/v2"
)

var _ http.RoundTripper = &BackendContext{}

type BackendContext struct {
	body hcl.Body
	rt   http.RoundTripper
}

func NewBackendContext(body hcl.Body, rt http.RoundTripper) http.RoundTripper {
	return &BackendContext{
		body: body,
		rt:   rt,
	}
}

func (b *BackendContext) RoundTrip(req *http.Request) (*http.Response, error) {
	return b.rt.RoundTrip(req.WithContext(context.WithValue(req.Context(), request.BackendContext, b.body)))
}
