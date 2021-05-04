package server

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/logging"
)

var (
	_ http.Flusher         = &RWWrapper{}
	_ http.Hijacker        = &RWWrapper{}
	_ http.ResponseWriter  = &RWWrapper{}
	_ logging.RecorderInfo = &RWWrapper{}
)

// RWWrapper wraps the <http.ResponseWriter>.
type RWWrapper struct {
	rw            http.ResponseWriter
	gz            *gzip.Writer
	headerBuffer  *bytes.Buffer
	httpStatus    []byte
	httpLineDelim []byte
	secureCookies string
	statusWritten bool
	// logging info
	statusCode      int
	rawBytesWritten int
	bytesWritten    int
}

// NewRWWrapper creates a new RWWrapper object.
func NewRWWrapper(rw http.ResponseWriter, useGZ bool, secureCookies string) *RWWrapper {
	w := &RWWrapper{
		rw:            rw,
		headerBuffer:  &bytes.Buffer{},
		secureCookies: secureCookies,
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
	l := len(p)
	w.rawBytesWritten += l
	if !w.statusWritten {
		if len(w.httpStatus) == 0 {
			w.httpStatus = p[:]
			// required for short writes without any additional header
			// to detect EOH chunk later on
			if l >= 2 {
				w.httpLineDelim = p[l-2 : l]
			}

			return l, nil
		}

		// End-of-header
		// http.Response.Write() EOH chunk is: '\r\n'
		if bytes.Equal(w.httpLineDelim, p) {
			reader := textproto.NewReader(bufio.NewReader(w.headerBuffer))
			header, _ := reader.ReadMIMEHeader()
			for k := range header {
				w.rw.Header()[k] = header.Values(k)
			}
			w.WriteHeader(w.parseStatusCode(w.httpStatus))
		}

		if l >= 2 {
			w.httpLineDelim = p[l-2 : l]
		}
		return w.headerBuffer.Write(p)
	}

	var n int
	var writeErr error
	if w.gz != nil {
		n, writeErr = w.gz.Write(p)
	} else {
		n, writeErr = w.rw.Write(p)
	}
	w.bytesWritten += n
	return n, writeErr
}

func (w *RWWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := w.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", w.rw)
	}
	return hijack.Hijack()
}

// Close closes the GZ writer.
func (w *RWWrapper) Close() {
	if w.gz != nil {
		_ = w.gz.Close()
	}
}

// Flush implements the <http.Flusher> interface.
func (w *RWWrapper) Flush() {
	if w.gz != nil {
		_ = w.gz.Flush()
	}

	if rw, ok := w.rw.(http.Flusher); ok {
		rw.Flush()
	}
}

// WriteHeader wraps the WriteHeader method of the <http.ResponseWriter>.
func (w *RWWrapper) WriteHeader(statusCode int) {
	if w.statusWritten {
		return
	}

	w.configureHeader()
	w.rw.WriteHeader(statusCode)
	w.statusWritten = true
	w.statusCode = statusCode
}

func (w *RWWrapper) configureHeader() {
	w.rw.Header().Set("Server", "couper.io")
	w.rw.Header().Add(transport.VaryHeader, transport.AcceptEncodingHeader)

	if w.gz != nil {
		w.rw.Header().Del(transport.ContentLengthHeader)
		w.rw.Header().Set(transport.ContentEncodingHeader, transport.GzipName)
	}

	if w.secureCookies == SecureCookiesStrip {
		stripSecureCookies(w.rw.Header())
	}
}

func (w *RWWrapper) parseStatusCode(p []byte) int {
	if len(p) < 12 {
		return 0
	}
	code, _ := strconv.Atoi(string(p[9:12]))
	return code
}

func (w *RWWrapper) StatusCode() int {
	return w.statusCode
}

func (w *RWWrapper) WrittenBytes() int {
	return w.bytesWritten
}

func (w *RWWrapper) ErrorHeader() string {
	return w.Header().Get(errors.HeaderErrorCode)
}
