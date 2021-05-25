package writer

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/avenga/couper/logging"
)

type writer interface {
	http.Flusher
	http.Hijacker
	http.ResponseWriter
}

var (
	_ writer               = &Response{}
	_ logging.RecorderInfo = &Response{}
)

// Response wraps the http.ResponseWriter.
type Response struct {
	rw            http.ResponseWriter
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

// NewResponseWriter creates a new Response object.
func NewResponseWriter(rw http.ResponseWriter, secureCookies string) *Response {
	w := &Response{
		rw:            rw,
		headerBuffer:  &bytes.Buffer{},
		secureCookies: secureCookies,
	}

	return w
}

// Header wraps the Header method of the <http.ResponseWriter>.
func (w *Response) Header() http.Header {
	return w.rw.Header()
}

// Write wraps the Write method of the <http.ResponseWriter>.
func (w *Response) Write(p []byte) (int, error) {
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

	n, writeErr := w.rw.Write(p)
	w.bytesWritten += n
	return n, writeErr
}

func (w *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := w.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", w.rw)
	}
	return hijack.Hijack()
}

// Flush implements the <http.Flusher> interface.
func (w *Response) Flush() {
	if rw, ok := w.rw.(http.Flusher); ok {
		rw.Flush()
	}
}

// WriteHeader wraps the WriteHeader method of the <http.ResponseWriter>.
func (w *Response) WriteHeader(statusCode int) {
	if w.statusWritten {
		return
	}

	w.configureHeader()
	w.rw.WriteHeader(statusCode)
	w.statusWritten = true
	w.statusCode = statusCode
}

func (w *Response) configureHeader() {
	w.rw.Header().Set("Server", "couper.io")

	if w.secureCookies == SecureCookiesStrip {
		stripSecureCookies(w.rw.Header())
	}
}

func (w *Response) parseStatusCode(p []byte) int {
	if len(p) < 12 {
		return 0
	}
	code, _ := strconv.Atoi(string(p[9:12]))
	return code
}

func (w *Response) StatusCode() int {
	return w.statusCode
}

func (w *Response) WrittenBytes() int {
	return w.bytesWritten
}
