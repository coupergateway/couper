package handler

import (
	"net/http"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
)

var _ http.Handler = &Error{}

type Error struct {
	kindContext map[string]hcl.Body
	template    *errors.Template
}

func NewErrorHandler(kindContext map[string]hcl.Body, tpl *errors.Template) *Error {
	return &Error{
		kindContext: kindContext,
		template:    tpl,
	}
}

func (e *Error) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err, ok := req.Context().Value(request.Error).(*errors.Error)
	if !ok { // all errors within this context should have this type, otherwise an implementation error
		e.template.ServeError(errors.Server).ServeHTTP(rw, req)
		return
	}

	if e.kindContext == nil { // nothing defined, just serve err with template
		e.template.ServeError(err).ServeHTTP(rw, req)
		return
	}

	for _, kind := range err.Kinds() {
		eh, defined := e.kindContext[kind]
		if !defined {
			continue
		}
		evalContext := req.Context().Value(eval.ContextType).(*eval.Context)
		resp, respErr := producer.NewResponse(req, eh, evalContext, err.HTTPStatus())
		if respErr != nil {
			e.template.ServeError(respErr).ServeHTTP(rw, req)
			return
		}
		eval.ApplyResponseContext(req.Context(), eh, resp)
		resp.Write(rw)
		return
	}

	// fallback with no matching error handler
	e.template.ServeError(err).ServeHTTP(rw, req)
}
