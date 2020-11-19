package server

import (
	"bytes"
	"compress/gzip"
	"net/http"

	"github.com/avenga/couper/handler"
)

var _ http.ResponseWriter = &RWWrapper{}

// RWWrapper wraps the <http.ResponseWriter>.
type RWWrapper struct {
	rw    http.ResponseWriter
	useGZ bool
}

// NewRWWrapper creates a new RWWrapper object.
func NewRWWrapper(rw http.ResponseWriter, useGZ bool) *RWWrapper {
	return &RWWrapper{
		rw:    rw,
		useGZ: useGZ,
	}
}

// Header wraps the Header method of the <http.ResponseWriter>.
func (w *RWWrapper) Header() http.Header {
	return w.rw.Header()
}

// Write wraps the Write method of the <http.ResponseWriter>.
func (w *RWWrapper) Write(p []byte) (int, error) {
	if w.useGZ {
		var buf bytes.Buffer

		gz := gzip.NewWriter(&buf)
		defer gz.Close()

		if n, err := gz.Write(p); err != nil {
			return n, err
		}

		gz.Close()

		p = buf.Bytes()
	}

	return w.rw.Write(p)
}

// WriteHeader wraps the WriteHeader method of the <http.ResponseWriter>.
func (w *RWWrapper) WriteHeader(statusCode int) {
	w.rw.Header().Set("Server", "couper.io")

	if w.useGZ {
		w.rw.Header().Del(handler.ContentLengthHeader)
		w.rw.Header().Set(handler.ContentEncodingHeader, handler.GzipName)
	}

	w.rw.WriteHeader(statusCode)
}
