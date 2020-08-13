package errors

import "net/http"

func httpStatus(code Code) int {
	switch code {
	case APIRouteNotFound, FilesRouteNotFound, RouteNotFound, SPARouteNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
