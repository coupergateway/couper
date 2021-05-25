package writer

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net"
	"net/http"
	"regexp"
)

const (
	AcceptEncodingHeader  = "Accept-Encoding"
	ContentEncodingHeader = "Content-Encoding"
	ContentLengthHeader   = "Content-Length"
	GzipName              = "gzip"
	VaryHeader            = "Vary"
)

var (
	clientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

	_ writer = &Gzip{}
)

type Gzip struct {
	enabled bool
	rw      http.ResponseWriter
	w       *gzip.Writer
	written int64
}

func NewGzipWriter(rw http.ResponseWriter, header http.Header) *Gzip {
	gw := gzip.NewWriter(rw)
	return &Gzip{
		enabled: clientSupportsGZ.MatchString(header.Get(AcceptEncodingHeader)),
		rw:      rw,
		w:       gw,
	}
}

func (g *Gzip) Write(p []byte) (n int, err error) {
	if g.enabled {
		return g.w.Write(p)
	}
	return g.rw.Write(p)
}

func (g *Gzip) Close() (err error) {
	if g.enabled && g.w != nil {
		err = g.w.Close()
	}
	return err
}

func (g *Gzip) Header() http.Header {
	return g.rw.Header()
}

func (g *Gzip) WriteHeader(statusCode int) {
	g.rw.Header().Add(VaryHeader, AcceptEncodingHeader)

	if g.enabled {
		g.rw.Header().Del(ContentLengthHeader)
		g.rw.Header().Set(ContentEncodingHeader, GzipName)
	}

	g.rw.WriteHeader(statusCode)
}

func (g *Gzip) Flush() {
	if g.enabled && g.w != nil {
		_ = g.w.Flush()
	}

	if rw, ok := g.rw.(http.Flusher); ok {
		rw.Flush()
	}
}

func (g *Gzip) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := g.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("can't switch protocols using non-Hijacker gzip writer type %T", g.rw)
	}
	return hijack.Hijack()
}

func ModifyAcceptEncoding(header http.Header) {
	if clientSupportsGZ.MatchString(header.Get(AcceptEncodingHeader)) {
		header.Set(AcceptEncodingHeader, GzipName)
	} else {
		header.Del(AcceptEncodingHeader)
	}
}
