package accesscontrol

import (
	"net/http"

	"github.com/avenga/couper/eval"
)

var _ AccessControl = ValidateFunc(func(_ *http.Request) error { return nil })

type ListItem struct {
	Func AccessControl
	Name string
}

func (i ListItem) Validate(req *http.Request) error {
	return i.Func.Validate(req)
}

type (
	Map  map[string]AccessControl
	List []ListItem
)

type ValidateFunc func(*http.Request) error

type AccessControl interface {
	Validate(req *http.Request) error
}

type ProtectedHandler interface {
	Child() http.Handler
}

func (f ValidateFunc) Validate(req *http.Request) error {
	if err := f(req); err != nil {
		return err
	}

	if evalCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		*req = *req.WithContext(evalCtx.WithClientRequest(req))
	}
	return nil
}

func (m Map) MustExist(name string) {
	if m == nil {
		panic("no accessControl configuration")
	}

	if _, ok := m[name]; !ok {
		panic("accessControl is not defined: " + name)
	}
}
