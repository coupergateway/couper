package access_control

import "net/http"

var (
	_ AccessControl = ValidateFunc(func(_ *http.Request) error {
		return nil
	})
	_ AccessControl = &JWT{}
)

type Map map[string]AccessControl
type List []AccessControl

type ValidateFunc func(*http.Request) error

type AccessControl interface {
	Validate(req *http.Request) error
}

func (f ValidateFunc) Validate(req *http.Request) error {
	return f(req)
}
