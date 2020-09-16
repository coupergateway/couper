package errors

import "net/http"

func httpStatus(code Code) int {
	switch code {
	case APIRouteNotFound, FilesRouteNotFound, RouteNotFound, SPARouteNotFound:
		return http.StatusNotFound
	case APIConnect:
		return http.StatusBadGateway
	case InvalidRequest:
		return http.StatusBadRequest
	case AuthorizationRequired:
		return http.StatusUnauthorized
	case AuthorizationFailed:
		return http.StatusForbidden
	case BasicAuthFailed:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
