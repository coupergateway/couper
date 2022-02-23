package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var defaultAllowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// https://datatracker.ietf.org/doc/html/rfc7231#section-4
// https://datatracker.ietf.org/doc/html/rfc7230#section-3.2.6
var methodRegExp = regexp.MustCompile("^[!#$%&'*+\\-.\\^_`|~0-9a-zA-Z]+$")

var _ http.Handler = &AllowedMethodsHandler{}

type AllowedMethodsHandler struct {
	allowedMethods    map[string]struct{}
	allowedHandler    http.Handler
	notAllowedHandler http.Handler
}

func NewAllowedMethodsHandler(allowedMethods []string, allowedHandler, notAllowedHandler http.Handler) (http.Handler, error) {
	amh := &AllowedMethodsHandler{
		allowedMethods:    make(map[string]struct{}),
		allowedHandler:    allowedHandler,
		notAllowedHandler: notAllowedHandler,
	}
	if allowedMethods == nil {
		allowedMethods = defaultAllowedMethods
	}
	for _, method := range allowedMethods {
		method = strings.TrimSpace(strings.ToUpper(method))
		if !methodRegExp.Match([]byte(method)) {
			return nil, fmt.Errorf("allowed_methods: invalid character in method %q", method)
		}
		amh.allowedMethods[method] = struct{}{}
	}

	return amh, nil
}

func (a *AllowedMethodsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if _, ok := a.allowedMethods[req.Method]; !ok {
		a.notAllowedHandler.ServeHTTP(rw, req)
		return
	}

	a.allowedHandler.ServeHTTP(rw, req)
}

func (a *AllowedMethodsHandler) Child() http.Handler {
	return a.allowedHandler
}
