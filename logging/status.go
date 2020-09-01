package logging

import (
	"net/http"
)

var _ http.ResponseWriter = &StatusReader{}

// StatusReader represents the StatusReader object.
type StatusReader struct {
	rw     http.ResponseWriter
	status int
}

// NewStatusReader creates a new StatusReader object.
func NewStatusReader(rw http.ResponseWriter) *StatusReader {
	return &StatusReader{rw: rw}
}

// Header wraps the Header method of the ResponseWriter.
func (sr *StatusReader) Header() http.Header {
	return sr.rw.Header()
}

// Write wraps the Write method of the ResponseWriter.
func (sr *StatusReader) Write(p []byte) (int, error) {
	return sr.rw.Write(p)
}

// WriteHeader wraps the WriteHeader method of the ResponseWriter.
func (sr *StatusReader) WriteHeader(statusCode int) {
	if sr.status == 0 {
		sr.status = statusCode
	}
	sr.rw.Header().Set("Server", "couper.io")
	sr.rw.WriteHeader(statusCode)
}
