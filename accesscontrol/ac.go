package accesscontrol

import (
	"net/http"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
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

type DisablePrivateCaching interface {
	DisablePrivateCaching() bool
}

type ProtectedHandler interface {
	Child() http.Handler
}

var _ AccessControl = ValidateFunc(func(_ *http.Request) error { return nil })

func (i ListItem) Kind() string {
	return i.kind
}

func (i ListItem) Label() string {
	return i.label
}

func (i ListItem) Validate(req *http.Request) error {
	if err := i.control.Validate(req); err != nil {
		if e, ok := err.(*errors.Error); ok {
			return e.Label(i.label)
		}
		return errors.AccessControl.Label(i.label).Kind(i.kind).With(err)
	}

	evalCtx := eval.ContextFromRequest(req)
	*req = *req.WithContext(evalCtx.WithClientRequest(req))

	return nil
}

func (i ListItem) ErrorHandler() http.Handler {
	return i.controlErrHandler
}

func (i ListItem) DisablePrivateCaching() bool {
	if c, ok := i.control.(DisablePrivateCaching); ok {
		return c.DisablePrivateCaching()
	}
	// not implemented, always disabled
	return true
}

func (f ValidateFunc) Validate(req *http.Request) error {
	return f(req)
}
