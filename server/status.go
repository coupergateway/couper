package server

import (
	"net/http"
)

var _ http.ResponseWriter = &StatusReader{}

type StatusReader struct {
	rw     http.ResponseWriter
	status int
}

func (sr *StatusReader) Header() http.Header {
	return sr.rw.Header()
}

func (sr *StatusReader) Write(p []byte) (int, error) {
	return sr.rw.Write(p)
}

func (sr *StatusReader) WriteHeader(statusCode int) {
	if sr.status == 0 {
		sr.status = statusCode
	}
	sr.rw.WriteHeader(statusCode)
}
