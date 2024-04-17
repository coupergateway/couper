package middleware

import (
	"context"
	"net/http"

	"github.com/coupergateway/couper/config/request"
)

type condition func(req *http.Request) error

func NewErrorHandler(condition condition, eh http.Handler) Next {
	return func(handler http.Handler) *NextHandler {
		return NewHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if err := condition(req); err != nil {
				*req = *req.WithContext(context.WithValue(req.Context(), request.Error, err))
				eh.ServeHTTP(rw, req)
				return
			}
			handler.ServeHTTP(rw, req)
		}), handler)
	}
}
