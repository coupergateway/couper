package middleware

import "net/http"

type NextHandler interface {
	ServeNextHTTP(http.ResponseWriter, http.Handler, *http.Request)
}
