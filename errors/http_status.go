package errors

import "net/http"

func httpStatus(code Code) int {
	switch code {
	case APIRouteNotFound, FilesRouteNotFound, RouteNotFound, SPARouteNotFound:
		return http.StatusNotFound
	case RequestError:
		return http.StatusBadRequest
	case AuthorizationRequired:
		return http.StatusUnauthorized
	case AuthorizationFailed:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
