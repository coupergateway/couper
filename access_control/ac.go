package access_control

import "net/http"

var (
	_ AccessControl = &JWT{}
)

type Map map[string]AccessControl
type List []AccessControl

type AccessControl interface {
	Validate(req *http.Request) error
}
