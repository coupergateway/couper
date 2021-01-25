package logging

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

var _ http.ResponseWriter = &Recorder{}
var _ http.Hijacker = &Recorder{}

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
	if sr.status == 0 {
		sr.WriteHeader(http.StatusOK)
	}
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

func (sr *Recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := sr.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", sr.rw)
	}
	return hijack.Hijack()
}
