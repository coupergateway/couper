package logging

import (
	"net/http"
)

var _ http.ResponseWriter = &Recorder{}

// Recorder represents the Recorder object.
type Recorder struct {
	rw           http.ResponseWriter
	status       int
	writtenBytes int
}

// NewStatusRecorder creates a new Recorder object.
func NewStatusRecorder(rw http.ResponseWriter) *Recorder {
	return &Recorder{rw: rw}
}

// Header wraps the Header method of the ResponseWriter.
func (sr *Recorder) Header() http.Header {
	return sr.rw.Header()
}

// Write wraps the Write method of the ResponseWriter.
func (sr *Recorder) Write(p []byte) (int, error) {
	i, err := sr.rw.Write(p)
	sr.writtenBytes += i
	return i, err
}

// WriteHeader wraps the WriteHeader method of the ResponseWriter.
func (sr *Recorder) WriteHeader(statusCode int) {
	if sr.status == 0 {
		sr.status = statusCode
	}
	sr.rw.WriteHeader(statusCode)
}
