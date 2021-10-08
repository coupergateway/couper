package middleware

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
)

type condition func(req *http.Request) error

func NewErrorHandler(condition condition, eh http.Handler) Next {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if err := condition(req); err != nil {
				eh.ServeHTTP(rw, req.WithContext(context.WithValue(req.Context(), request.Error, err)))
				return
			}
			handler.ServeHTTP(rw, req)
		})
	}
}
