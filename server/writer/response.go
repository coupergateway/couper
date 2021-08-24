package writer

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/logging"
	"github.com/hashicorp/hcl/v2"
)

type writer interface {
	http.Flusher
	http.Hijacker
	http.ResponseWriter
}

type modifier interface {
	AddModifier(*eval.Context, []hcl.Body)
}

var (
	_ writer               = &Response{}
	_ modifier             = &Response{}
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
	// modifier
	evalCtx  *eval.Context
	modifier []hcl.Body
}

// NewResponseWriter creates a new Response object.
func NewResponseWriter(rw http.ResponseWriter, secureCookies string) *Response {
	return &Response{
		rw:            rw,
		headerBuffer:  &bytes.Buffer{},
		secureCookies: secureCookies,
	}
}

// Header wraps the Header method of the <http.ResponseWriter>.
func (r *Response) Header() http.Header {
	return r.rw.Header()
}

// Write wraps the Write method of the <http.ResponseWriter>.
func (r *Response) Write(p []byte) (int, error) {
	l := len(p)
	r.rawBytesWritten += l
	if !r.statusWritten {
		if len(r.httpStatus) == 0 {
			r.httpStatus = p[:]
			// required for short writes without any additional header
			// to detect EOH chunk later on
			if l >= 2 {
				r.httpLineDelim = p[l-2 : l]
			}

			return l, nil
		}

		// End-of-header
		// http.Response.Write() EOH chunk is: '\r\n'
		if bytes.Equal(r.httpLineDelim, p) {
			reader := textproto.NewReader(bufio.NewReader(r.headerBuffer))
			header, _ := reader.ReadMIMEHeader()
			for k := range header {
				r.rw.Header()[k] = header.Values(k)
			}
			r.WriteHeader(r.parseStatusCode(r.httpStatus))
		}

		if l >= 2 {
			r.httpLineDelim = p[l-2 : l]
		}
		return r.headerBuffer.Write(p)
	}

	n, writeErr := r.rw.Write(p)
	r.bytesWritten += n
	return n, writeErr
}

func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := r.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", r.rw)
	}

	conn, rw, err := hijack.Hijack()
	if err != nil {
		return nil, nil, err
	}

	return conn, rw, err
}

// Flush implements the <http.Flusher> interface.
func (r *Response) Flush() {
	if rw, ok := r.rw.(http.Flusher); ok {
		rw.Flush()
	}
}

// WriteHeader wraps the WriteHeader method of the <http.ResponseWriter>.
func (r *Response) WriteHeader(statusCode int) {
	if r.statusWritten {
		return
	}

	r.configureHeader()
	r.applyModifier()

	if statusCode == 0 {
		r.rw.Header().Set(errors.HeaderErrorCode, errors.Server.Error())
		r.rw.WriteHeader(errors.Server.HTTPStatus())
	} else {
		r.rw.WriteHeader(statusCode)
	}
	r.statusWritten = true
	r.statusCode = statusCode
}

func (r *Response) configureHeader() {
	r.rw.Header().Set("Server", "couper.io")

	if r.secureCookies == SecureCookiesStrip {
		stripSecureCookies(r.rw.Header())
	}
}

func (r *Response) parseStatusCode(p []byte) int {
	if len(p) < 12 {
		return 0
	}
	code, _ := strconv.Atoi(string(p[9:12]))
	return code
}

func (r *Response) StatusCode() int {
	return r.statusCode
}

func (r *Response) WrittenBytes() int {
	return r.bytesWritten
}

func (r *Response) AddModifier(evalCtx *eval.Context, modifier []hcl.Body) {
	r.evalCtx = evalCtx
	r.modifier = append(r.modifier, modifier...)
}

func (r *Response) applyModifier() {
	if r.evalCtx == nil || r.modifier == nil {
		return
	}

	for _, body := range r.modifier {
		eval.ApplyResponseHeaderOps(r.evalCtx, body, r.Header())
	}
}
