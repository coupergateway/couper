package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
)

var _ http.Handler = &CustomLogs{}

type CustomLogs struct {
	bodies      []hcl.Body
	handlerName string
	next        http.Handler
}

func NewCustomLogsHandler(bodies []hcl.Body, next http.Handler, handlerName string) http.Handler {
	return NewHandler(&CustomLogs{
		bodies:      bodies,
		handlerName: handlerName,
		next:        next,
	}, next)
}

func (c *CustomLogs) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var bodies []hcl.Body

	if b := req.Context().Value(request.LogCustomAccess); b != nil {
		bodies = b.([]hcl.Body)
	}

	ctx := context.WithValue(req.Context(), request.LogCustomAccess, append(bodies, c.bodies...))
	ctx = context.WithValue(ctx, request.LogCustomEvalResult, make(chan *eval.Context, 2))
	*req = *req.WithContext(ctx)

	c.next.ServeHTTP(rw, req)
}

func (c *CustomLogs) String() string {
	if hs, stringer := c.next.(fmt.Stringer); stringer {
		return hs.String()
	}

	return c.handlerName
}
