package middleware

import "net/http"

type Next func(http.Handler) *NextHandler

type NextHandler struct {
	handler, next http.Handler
}

func NewHandler(handler, next http.Handler) *NextHandler {
	return &NextHandler{
		handler: handler,
		next:    next,
	}
}

func (n *NextHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	n.handler.ServeHTTP(rw, req)
}

func (n *NextHandler) Child() http.Handler {
	return n.next
}
