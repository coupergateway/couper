package accesscontrol

import (
	"fmt"
	"net/http"

	"github.com/avenga/couper/eval"
)

type (
	Map          map[string]AccessControl
	ValidateFunc func(*http.Request) error

	List     []ListItem
	ListItem struct {
		Func AccessControl
		Name string
	}
)

type AccessControl interface {
	Validate(req *http.Request) error
}

type ProtectedHandler interface {
	Child() http.Handler
}

var _ AccessControl = ValidateFunc(func(_ *http.Request) error { return nil })

func (i ListItem) Validate(req *http.Request) error {
	return i.Func.Validate(req)
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
