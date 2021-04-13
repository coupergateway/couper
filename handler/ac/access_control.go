package ac

import (
	"net/http"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var (
	_ http.Handler                   = &AccessControl{}
	_ errors.ErrorTemplate           = &AccessControl{}
	_ accesscontrol.ProtectedHandler = &AccessControl{}
)

type AccessControl struct {
	acl       accesscontrol.List
	errorTpl  *errors.Template
	protected http.Handler
}

func NewAccessControl(protected http.Handler, errTpl *errors.Template, list accesscontrol.List) *AccessControl {
	return &AccessControl{
		acl:       list,
		errorTpl:  errTpl,
		protected: protected,
	}
}

func (a *AccessControl) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, control := range a.acl {
		if err := control.Validate(req); err != nil {
			var code errors.Code
			if authError, ok := err.(*accesscontrol.BasicAuthError); ok {
				code = errors.BasicAuthFailed
				wwwAuthenticateValue := "Basic"
				if authError.Realm != "" {
					wwwAuthenticateValue += " realm=" + authError.Realm
				}
				rw.Header().Set("WWW-Authenticate", wwwAuthenticateValue)
			} else {
				switch err {
				case accesscontrol.ErrorNotConfigured:
					code = errors.Configuration
				case accesscontrol.ErrorEmptyToken:
					code = errors.AuthorizationRequired
				default:
					code = errors.AuthorizationFailed
				}
			}
			if ctx, ok := req.Context().Value(request.AccessControl).(*AccessControlContext); ok {
				ctx.error = err
				ctx.name = control.Name
			}
			a.errorTpl.ServeError(code).ServeHTTP(rw, req)
			return
		}
	}
	a.protected.ServeHTTP(rw, req)
}

func (a *AccessControl) Child() http.Handler {
	return a.protected
}

func (a *AccessControl) Template() *errors.Template {
	return a.errorTpl
}

func (a *AccessControl) String() string {
	if h, ok := a.protected.(interface{ String() string }); ok {
		return h.String()
	}
	return "AccessControl"
}
