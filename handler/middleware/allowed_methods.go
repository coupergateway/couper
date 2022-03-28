package middleware

import (
	"net/http"
	"strings"
)

var _ http.Handler = &AllowedMethodsHandler{}

type AllowedMethodsHandler struct {
	allowedMethods    map[string]struct{}
	allowedHandler    http.Handler
	notAllowedHandler http.Handler
}

type methodAllowedFunc func(string) bool

func NewAllowedMethodsHandler(allowedMethods, defaultAllowedMethods []string, allowedHandler, notAllowedHandler http.Handler) *AllowedMethodsHandler {
	amh := &AllowedMethodsHandler{
		allowedMethods:    make(map[string]struct{}),
		allowedHandler:    allowedHandler,
		notAllowedHandler: notAllowedHandler,
	}
	if allowedMethods == nil && defaultAllowedMethods != nil {
		allowedMethods = defaultAllowedMethods
	}
	for _, method := range allowedMethods {
		if method == "*" {
			if defaultAllowedMethods == nil {
				continue
			}
			for _, m := range defaultAllowedMethods {
				amh.allowedMethods[m] = struct{}{}
			}
		} else {
			method = strings.ToUpper(method)
			amh.allowedMethods[method] = struct{}{}
		}
	}

	return amh
}

func (a *AllowedMethodsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if _, ok := a.allowedMethods[req.Method]; !ok {
		a.notAllowedHandler.ServeHTTP(rw, req)
		return
	}

	a.allowedHandler.ServeHTTP(rw, req)
}

func (a *AllowedMethodsHandler) MethodAllowed(method string) bool {
	method = strings.TrimSpace(strings.ToUpper(method))
	_, ok := a.allowedMethods[method]
	return ok
}

func (a *AllowedMethodsHandler) Child() http.Handler {
	return a.allowedHandler
}
