package accesscontrol

import "net/http"

var _ AccessControl = ValidateFunc(func(_ *http.Request) error { return nil })

const ContextAccessControlKey = "access_controls"

type (
	Map  map[string]AccessControl
	List []AccessControl
)

type ValidateFunc func(*http.Request) error

type AccessControl interface {
	Validate(req *http.Request) error
}

type ProtectedHandler interface {
	Child() http.Handler
}

func (f ValidateFunc) Validate(req *http.Request) error {
	return f(req)
}

func (m Map) MustExist(name string) {
	if m == nil {
		panic("no accessControl configuration")
	}

	if _, ok := m[name]; !ok {
		panic("accessControl is not defined: " + name)
	}
}
