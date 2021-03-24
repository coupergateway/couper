package handler

import (
	"net/http"

	"github.com/avenga/couper/errors"
)

var _ http.Handler = &Error{}

type Error struct {
	template *errors.Template
}

func NewErrorHandler(tpl *errors.Template) *Error {
	return &Error{
		template: tpl,
	}
}

func (e *Error) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	panic("implement me")
}
