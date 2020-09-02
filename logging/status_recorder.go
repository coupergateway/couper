package logging

import (
	"net/http"
)

var _ http.ResponseWriter = &StatusRecorder{}

// StatusRecorder represents the StatusRecorder object.
type StatusRecorder struct {
	rw     http.ResponseWriter
	status int
}

// NewStatusRecorder creates a new StatusRecorder object.
func NewStatusRecorder(rw http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{rw: rw}
}

// Header wraps the Header method of the ResponseWriter.
func (sr *StatusRecorder) Header() http.Header {
	return sr.rw.Header()
}

// Write wraps the Write method of the ResponseWriter.
func (sr *StatusRecorder) Write(p []byte) (int, error) {
	return sr.rw.Write(p)
}

// WriteHeader wraps the WriteHeader method of the ResponseWriter.
func (sr *StatusRecorder) WriteHeader(statusCode int) {
	if sr.status == 0 {
		sr.status = statusCode
	}
	sr.rw.Header().Set("Server", "couper.io")
	sr.rw.WriteHeader(statusCode)
}
