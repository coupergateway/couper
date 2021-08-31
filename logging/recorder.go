package logging

import (
	"net/http"
)

var _ http.ResponseWriter = &StatusRecorder{}
var _ RecorderInfo = &StatusRecorder{}

type StatusRecorder struct {
	rw            http.ResponseWriter
	statusCode    int
	statusWritten bool
	bytes         int
}

func NewStatusRecorder(rw http.ResponseWriter) http.ResponseWriter {
	return &StatusRecorder{rw: rw}
}

func (s *StatusRecorder) StatusCode() int {
	return s.statusCode
}

func (s *StatusRecorder) WrittenBytes() int {
	return s.bytes
}

func (s *StatusRecorder) Header() http.Header {
	return s.rw.Header()
}

func (s *StatusRecorder) Write(bytes []byte) (int, error) {
	i, err := s.rw.Write(bytes)
	if s.statusWritten {
		s.bytes += i
	}
	return i, err
}

func (s *StatusRecorder) WriteHeader(statusCode int) {
	if s.statusWritten {
		return
	}
	s.statusCode = statusCode
	s.statusWritten = true
}
