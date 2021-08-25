package writer

import (
	"bufio"
	"bytes"
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

	minCompressBodyLength = 60
)

var (
	clientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

	_ writer = &Gzip{}
)

type Gzip struct {
	buffer     *bytes.Buffer
	enabled    bool
	hijacked   bool
	headerSent bool
	statusCode int
	writeErr   error
	rw         http.ResponseWriter
	w          *gzip.Writer
}

func NewGzipWriter(rw http.ResponseWriter, header http.Header) *Gzip {
	return &Gzip{
		buffer:  bytes.NewBuffer(nil),
		enabled: clientSupportsGZ.MatchString(header.Get(AcceptEncodingHeader)),
		rw:      rw,
		w:       gzip.NewWriter(rw),
	}
}

// Write fills a small buffer first to determine if a compression is required or not.
func (g *Gzip) Write(p []byte) (n int, err error) {
	b := p[:]
	bytesLen := len(p)
	bufLen := g.buffer.Len()

	if bufLen < minCompressBodyLength {
		limit := minCompressBodyLength - bufLen

		if bytesLen < limit {
			return g.buffer.Write(b)
		}

		// Fill the buffer at least to minCompressBodyLength size.
		if _, err = g.buffer.Write(b); err != nil {
			return 0, err
		}

		b = g.buffer.Bytes()
	}

	g.writeHeader()

	n, err = g.write(b)
	if err != nil {
		return n, err
	} else if bufLen < minCompressBodyLength && bytesLen != (n-bufLen) {
		return 0, fmt.Errorf("invalid write result")
	}

	return bytesLen, err
}

func (g *Gzip) write(p []byte) (n int, err error) {
	if g.enabled {
		return g.w.Write(p)
	}
	return g.rw.Write(p)
}

func (g *Gzip) Close() (err error) {
	if g.writeErr != nil {
		return g.writeErr
	}

	if g.buffer.Len() < minCompressBodyLength {
		g.enabled = false
		g.writeHeader()

		_, err = g.write(g.buffer.Bytes())
		if err != nil {
			return err
		}
	}

	g.writeHeader()

	if g.enabled && g.w != nil {
		err = g.w.Close()
	}

	return err
}

func (g *Gzip) Header() http.Header {
	return g.rw.Header()
}

func (g *Gzip) WriteHeader(statusCode int) {
	g.statusCode = statusCode
}

func (g *Gzip) writeHeader() {
	if g.headerSent {
		return
	}

	g.headerSent = true

	if g.buffer.Len() >= minCompressBodyLength {
		g.rw.Header().Add(VaryHeader, AcceptEncodingHeader)
	}

	if g.enabled {
		g.rw.Header().Del(ContentLengthHeader)
		g.rw.Header().Set(ContentEncodingHeader, GzipName)
	}

	if !g.hijacked {
		g.rw.WriteHeader(g.statusCode)
	}
}

func (g *Gzip) Flush() {
	if l := g.buffer.Len(); l < minCompressBodyLength {
		// We have to wait for minCompressBodyLength bytes to be
		// able to determine, if we enable GZIP compression or not.
		return
	}

	g.writeHeader()

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

	g.enabled = false
	g.hijacked = true

	return hijack.Hijack()
}

func ModifyAcceptEncoding(header http.Header) {
	if clientSupportsGZ.MatchString(header.Get(AcceptEncodingHeader)) {
		header.Set(AcceptEncodingHeader, GzipName)
	} else {
		header.Del(AcceptEncodingHeader)
	}
}
