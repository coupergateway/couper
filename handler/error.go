package handler

import (
	"net/http"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/logging"
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
		e.template.WithError(errors.Server).ServeHTTP(rw, req)
		return
	}

	if e.kindsHandler == nil { // nothing defined, just serve err with template
		e.template.WithError(err).ServeHTTP(rw, req)
		return
	}

	for _, kind := range err.Kinds() {
		ep, defined := e.kindsHandler[kind]
		if !defined {
			continue
		}

		if eph, ek := ep.(*Endpoint); ek {
			// Custom log context is set on configuration load, however which error events got thrown
			// during runtime is unknown at this point, so we must update the hcl context here and
			// with the wildcard cases below.
			logging.UpdateCustomAccessLogContext(req, eph.BodyContext())
		}

		ep.ServeHTTP(rw, req)
		return
	}

	if ep, defined := e.kindsHandler[errors.Wildcard]; defined {
		if eph, ek := ep.(*Endpoint); ek {
			logging.UpdateCustomAccessLogContext(req, eph.BodyContext())
		}
		ep.ServeHTTP(rw, req)
		return
	}

	// fallback with no matching error handler
	e.template.WithError(err).ServeHTTP(rw, req)
}
