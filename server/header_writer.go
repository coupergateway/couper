package server

import (
	"net/http"
)

var _ http.ResponseWriter = &HeaderWriter{}

// ServerHeaderWriter ensures the server header value to be couper.io.
type HeaderWriter struct {
	rw http.ResponseWriter
}

// NewHeaderWriter creates a new HeaderWriter object.
func NewHeaderWriter(rw http.ResponseWriter) *HeaderWriter {
	return &HeaderWriter{rw: rw}
}

// Header wraps the Header method of the ResponseWriter.
func (sr *HeaderWriter) Header() http.Header {
	return sr.rw.Header()
}

// Write wraps the Write method of the ResponseWriter.
func (sr *HeaderWriter) Write(p []byte) (int, error) {
	return sr.rw.Write(p)
}

// WriteHeader wraps the WriteHeader method of the ResponseWriter.
func (sr *HeaderWriter) WriteHeader(statusCode int) {
	sr.rw.Header().Set("Server", "couper.io")
	sr.rw.WriteHeader(statusCode)
}
