package transport

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
)

var (
	_ http.Hijacker       = &Recorder{}
	_ http.ResponseWriter = &Recorder{}
	_ io.Reader           = &Recorder{}
)

// Recorder buffers a proxy response and assume that calls to Write
// contains body content only. If this recorder is used for writing
// a textproto http response, Write method must implement status and header reading.
type Recorder struct {
	body       *bytes.Buffer
	err        error
	header     http.Header
	rw         http.Hijacker
	statusCode int
}

func NewRecorder(rw http.Hijacker) *Recorder {
	return &Recorder{
		body: &bytes.Buffer{},
		rw:   rw,
	}
}

func (r *Recorder) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *Recorder) Write(p []byte) (int, error) {
	return r.body.Write(p)
}

func (r *Recorder) WriteHeader(statusCode int) {
	if r.statusCode == 0 {
		r.statusCode = statusCode
	}
}

func (r *Recorder) Read(p []byte) (n int, err error) {
	return r.body.Read(p)
}

func (r *Recorder) Response(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Body:       io.NopCloser(r.body),
		Header:     r.Header().Clone(),
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Request:    req,
		Status:     http.StatusText(r.statusCode),
		StatusCode: r.statusCode,
	}, r.err
}

func (r *Recorder) SetError(err error) {
	r.err = err
}

func (r *Recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if r.rw == nil {
		return nil, nil, fmt.Errorf("recorder type error: hijacker is nil")
	}
	return r.rw.Hijack()
}
