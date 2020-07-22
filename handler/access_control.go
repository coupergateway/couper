package handler

import (
	"net/http"

	ac "go.avenga.cloud/couper/gateway/access_control"
)

type AccessControl struct {
	ac        ac.List
	protected http.Handler
}

func NewAccessControl(protected http.Handler, list ...ac.AccessControl) *AccessControl {
	return &AccessControl{ac: list, protected: protected}
}

func (a *AccessControl) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, control := range a.ac {
		if err := control.Validate(req); err != nil { // TODO: statusCode returned by validate() ?
			rw.WriteHeader(http.StatusForbidden)
			rw.Write([]byte("<html><body><pre>" + err.Error() + "</pre></body></html>\n")) // TODO: json based on api
			return
		}
	}
	a.protected.ServeHTTP(rw, req)
}

func (a *AccessControl) String() string {
	if h, ok := a.protected.(interface{ String() string }); ok {
		return h.String()
	}
	return "AccessControl"
}
