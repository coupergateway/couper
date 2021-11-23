package middleware

import (
	"context"
	"fmt"
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
	bodies      []hcl.Body
	handlerName string
	next        http.Handler
}

func NewCustomLogsHandler(bodies []hcl.Body, next http.Handler, handlerName string) http.Handler {
	return &CustomLogs{
		bodies:      bodies,
		handlerName: handlerName,
		next:        next,
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

func (c *CustomLogs) String() string {
	if hs, stringer := c.next.(fmt.Stringer); stringer {
		return hs.String()
	}

	return c.handlerName
}
