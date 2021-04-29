package handler

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var _ http.Handler = &Error{}

type Error struct {
	kindsHandler map[string]http.Handler
	template     *errors.Template
}

func NewErrorHandler(kindsHandler map[string]http.Handler, errTpl *errors.Template) *Error {
	return &Error{
		kindsHandler: kindsHandler,
		template:     errTpl,
	}
}

func (e *Error) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err, ok := req.Context().Value(request.Error).(*errors.Error)
	if !ok { // all errors within this context should have this type, otherwise an implementation error
		e.template.ServeError(errors.Server).ServeHTTP(rw, req)
		return
	}

	if kinds := err.Kinds(); len(kinds) > 0 {
		*req = *req.WithContext(context.WithValue(req.Context(), request.ErrorKind, kinds[0]))
	}

	if e.kindsHandler == nil { // nothing defined, just serve err with template
		e.template.ServeError(err).ServeHTTP(rw, req)
		return
	}

	for _, kind := range err.Kinds() {
		ep, defined := e.kindsHandler[kind]
		if !defined {
			continue
		}
		ep.ServeHTTP(rw, req)
		return
	}

	if ep, defined := e.kindsHandler[errors.Wildcard]; defined {
		ep.ServeHTTP(rw, req)
		return
	}

	// fallback with no matching error handler
	e.template.ServeError(err).ServeHTTP(rw, req)
}
