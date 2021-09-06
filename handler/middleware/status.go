package middleware

import (
	"net/http"

	"github.com/avenga/couper/logging"
)

func NewStatusRecordHandler() Next {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			statusRW := logging.NewStatusRecorder(rw)
			handler.ServeHTTP(statusRW, r)
		})
	}
}
