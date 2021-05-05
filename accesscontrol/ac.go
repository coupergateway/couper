package accesscontrol

import (
	"net/http"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

type (
	Map          map[string]AccessControl
	ValidateFunc func(*http.Request) error

	List     []*ListItem
	ListItem struct {
		control           AccessControl
		controlErrHandler http.Handler
		kind              string
		label             string
	}
)

func NewItem(nameLabel string, control AccessControl, errHandler http.Handler) *ListItem {
	return &ListItem{
		control:           control,
		controlErrHandler: errHandler,
		kind:              errors.TypeToSnake(control),
		label:             nameLabel,
	}
}

type AccessControl interface {
	Validate(req *http.Request) error
}

type ProtectedHandler interface {
	Child() http.Handler
}

var _ AccessControl = ValidateFunc(func(_ *http.Request) error { return nil })

func (i ListItem) Validate(req *http.Request) error {
	if err := i.control.Validate(req); err != nil {
		if e, ok := err.(*errors.Error); ok {
			return e.Label(i.label)
		}
		return errors.AccessControl.Label(i.label).Kind(i.kind).With(err)
	}

	if evalCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		*req = *req.WithContext(evalCtx.WithClientRequest(req))
	}

	return nil
}

func (i ListItem) ErrorHandler() http.Handler {
	return i.controlErrHandler
}

func (f ValidateFunc) Validate(req *http.Request) error {
	return f(req)
}
