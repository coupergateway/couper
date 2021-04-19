package accesscontrol

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"

	"github.com/avenga/couper/errors"

	"github.com/avenga/couper/eval"
)

type (
	Map          map[string]AccessControl
	ValidateFunc func(*http.Request) error

	List     []*ListItem
	ListItem struct {
		control AccessControl
		kind    string
		label   string
	}
)

func NewItem(nameLabel string, control AccessControl) *ListItem {
	return &ListItem{
		control: control,
		kind:    TypeToSnake(control),
		label:   nameLabel,
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
	err := i.control.Validate(req)
	if gerr, ok := err.(*errors.Error); ok {
		return gerr.Label(i.label).PrefixKind(i.kind)
	}
	return errors.AccessControl.Label(i.label).Kind(i.kind).With(err)
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

func TypeToSnake(t interface{}) string {
	typeStr := reflect.TypeOf(t).String()
	if strings.Contains(typeStr, ".") { // package name removal
		typeStr = strings.Split(typeStr, ".")[1]
	}
	var result []rune
	for i, r := range typeStr {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}

	return string(result)
}
