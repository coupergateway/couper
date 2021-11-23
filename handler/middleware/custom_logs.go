package middleware

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/hashicorp/hcl/v2"
)

type HasResponse interface {
	HasResponse(req *http.Request) bool
}

var (
	_ http.Handler = &CustomLogs{}
	_ HasResponse  = &CustomLogs{}
)

type CustomLogs struct {
	next   http.Handler
	bodies []hcl.Body
}

func NewCustomLogsHandler(bodies []hcl.Body, next http.Handler) http.Handler {
	return &CustomLogs{
		bodies: bodies,
		next:   next,
	}
}

func (c *CustomLogs) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var bodies []hcl.Body

	if b := req.Context().Value(request.AccessLogFields); b != nil {
		bodies = b.([]hcl.Body)
	}

	ctx := context.WithValue(req.Context(), request.AccessLogFields, append(bodies, c.bodies...))
	*req = *req.WithContext(ctx)

	c.next.ServeHTTP(rw, req)
}

func (c *CustomLogs) HasResponse(req *http.Request) bool {
	if fh, ok := c.next.(HasResponse); ok {
		return fh.HasResponse(req)
	}

	return false
}
