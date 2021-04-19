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
	errKind, ok := req.Context().Value(request.ErrorKind).(error)
	if !ok {
		e.template.ServeError(errors.Server).ServeHTTP(rw, req)
		return
	}

	gerr := errKind.(errors.GoError)
	if eh, defined := e.kindContext[gerr.Type()]; defined {
		evalContext := req.Context().Value(eval.ContextType).(*eval.Context)
		resp, err := producer.NewResponse(req, eh, evalContext, gerr.GoStatus())
		if err != nil {
			e.template.ServeError(err).ServeHTTP(rw, req)
			return
		}
		eval.ApplyResponseContext(req.Context(), eh, resp)
		resp.Write(rw)
		return
	}

	e.template.ServeError(errKind).ServeHTTP(rw, req)
}
