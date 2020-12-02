package server

import (
	"compress/gzip"
	"net/http"

	"github.com/avenga/couper/handler"
)

var (
	_ http.ResponseWriter = &RWWrapper{}
	_ http.Flusher        = &RWWrapper{}
)

// RWWrapper wraps the <http.ResponseWriter>.
type RWWrapper struct {
	rw http.ResponseWriter
	gz *gzip.Writer
}

// NewRWWrapper creates a new RWWrapper object.
func NewRWWrapper(rw http.ResponseWriter, useGZ bool) *RWWrapper {
	w := &RWWrapper{
		rw: rw,
	}

	if useGZ {
		w.gz = gzip.NewWriter(rw)
	}

	return w
}

// Header wraps the Header method of the <http.ResponseWriter>.
func (w *RWWrapper) Header() http.Header {
	return w.rw.Header()
}

// Write wraps the Write method of the <http.ResponseWriter>.
func (w *RWWrapper) Write(p []byte) (int, error) {
	if w.gz != nil {
		return w.gz.Write(p)
	}

	return w.rw.Write(p)
}

// Close closes the GZ writer.
func (w *RWWrapper) Close() {
	if w.gz != nil {
		w.gz.Close()
	}
}

// Flush implements the <http.Flusher> interface.
func (w *RWWrapper) Flush() {
	if w.gz != nil {
		w.gz.Flush()
	}

	if rw, ok := w.rw.(http.Flusher); ok {
		rw.Flush()
	}
}

// WriteHeader wraps the WriteHeader method of the <http.ResponseWriter>.
func (w *RWWrapper) WriteHeader(statusCode int) {
	w.rw.Header().Set("Server", "couper.io")
	w.rw.Header().Add(handler.VaryHeader, handler.AcceptEncodingHeader)

	if w.gz != nil {
		w.rw.Header().Del(handler.ContentLengthHeader)
		w.rw.Header().Set(handler.ContentEncodingHeader, handler.GzipName)
	}

	w.rw.WriteHeader(statusCode)
}
