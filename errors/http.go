package errors

import (
	"fmt"
	"net/http"
)

func httpStatus(code Code) int {
	switch code {
	case APIRouteNotFound, FilesRouteNotFound, RouteNotFound, SPARouteNotFound:
		return http.StatusNotFound
	case APIConnect:
		return http.StatusBadGateway
	case APIReqBodySizeExceeded:
		return http.StatusRequestEntityTooLarge
	case InvalidRequest:
		return http.StatusBadRequest
	case AuthorizationRequired, BasicAuthFailed:
		return http.StatusUnauthorized
	case AuthorizationFailed:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func formatHeader(code Code) string {
	return fmt.Sprintf("%d - %q", code, code)
}

func SetHeader(rw http.ResponseWriter, code Code) {
	rw.Header().Set(HeaderErrorCode, formatHeader(code))
}
