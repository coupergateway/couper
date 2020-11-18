package server

import (
	"bytes"
	"compress/gzip"
	"net/http"
)

var _ http.ResponseWriter = &BodyZipper{}

// BodyZipper compresses the response body by the gzip compression.
type BodyZipper struct {
	rw               http.ResponseWriter
	clientSupportsGZ bool
}

// NewBodyZipper creates a new BodyZipper object.
func NewBodyZipper(rw http.ResponseWriter, clientSupportsGZ bool) *BodyZipper {
	return &BodyZipper{
		rw:               rw,
		clientSupportsGZ: clientSupportsGZ,
	}
}

// Header wraps the Header method of the ResponseWriter.
func (bz *BodyZipper) Header() http.Header {
	return bz.rw.Header()
}

// Write wraps the Write method of the ResponseWriter.
func (bz *BodyZipper) Write(p []byte) (int, error) {
	if bz.clientSupportsGZ {
		var buf bytes.Buffer

		gz := gzip.NewWriter(&buf)
		defer gz.Close()

		if n, err := gz.Write(p); err != nil {
			return n, err
		}

		p = buf.Bytes()
	}

	return bz.rw.Write(p)
}

// WriteHeader wraps the WriteHeader method of the ResponseWriter.
func (bz *BodyZipper) WriteHeader(statusCode int) {
	bz.rw.WriteHeader(statusCode)
}
