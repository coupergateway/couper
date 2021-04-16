package ac

import (
	"context"
	"net/http"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
)

var (
	_ http.Handler                   = &AccessControl{}
	_ accesscontrol.ProtectedHandler = &AccessControl{}
)

type AccessControl struct {
	acl          accesscontrol.List
	errorHandler http.Handler
	protected    http.Handler
}

func NewAccessControl(protected, errorHandler http.Handler, list accesscontrol.List) *AccessControl {
	return &AccessControl{
		acl:          list,
		errorHandler: errorHandler,
		protected:    protected,
	}
}

func (a *AccessControl) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, control := range a.acl {
		if err := control.Validate(req); err != nil {
			// TODO: register basic auth error handler
			//if authError, ok := err.(*accesscontrol.BasicAuthError); ok {
			//	wwwAuthenticateValue := "Basic"
			//	if authError.Realm != "" {
			//		wwwAuthenticateValue += " realm=" + authError.Realm
			//	}
			//	rw.Header().Set("WWW-Authenticate", wwwAuthenticateValue)
			//}

			*req = *req.WithContext(context.WithValue(req.Context(), request.ErrorKind, err))

			a.errorHandler.ServeHTTP(rw, req)
			return
		}
	}
	a.protected.ServeHTTP(rw, req)
}

func (a *AccessControl) Child() http.Handler {
	return a.protected
}

func (a *AccessControl) String() string {
	if h, ok := a.protected.(interface{ String() string }); ok {
		return h.String()
	}
	return "AccessControl"
}
