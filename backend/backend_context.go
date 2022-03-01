package backend

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/hashicorp/hcl/v2"
)

var _ http.RoundTripper = &Context{}

type Context struct {
	body hcl.Body
	rt   http.RoundTripper
}

func NewContext(body hcl.Body, rt http.RoundTripper) http.RoundTripper {
	return &Context{
		body: body,
		rt:   rt,
	}
}

func (b *Context) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), request.BackendParams, b.body)
	return b.rt.RoundTrip(req.WithContext(ctx))
}
