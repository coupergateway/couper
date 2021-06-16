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

	minCompressBodyLenght = 60
)

var (
	clientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

	_ writer = &Gzip{}
)

type Gzip struct {
	buffer     *bytes.Buffer
	enabled    bool
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

func (g *Gzip) Write(p []byte) (n int, err error) {
	bytesLen := len(p)
	bufLen := g.buffer.Len()

	if bufLen < minCompressBodyLenght {
		cap := minCompressBodyLenght - bufLen

		if bytesLen < cap {
			return g.buffer.Write(p)
		}

		// Fill the buffer at least to minCompressBodyLenght size.
		_, err := g.buffer.Write(p)
		if err != nil {
			return 0, err
		}

		p = g.buffer.Bytes()
	}

	g.writeHeader()

	l, err := g.write(p)
	if err != nil {
		return l, err
	} else if bufLen < minCompressBodyLenght && bytesLen != (l-bufLen) {
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

	if g.buffer.Len() < minCompressBodyLenght {
		g.enabled = false
		g.writeHeader()

		_, err := g.write(g.buffer.Bytes())
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

	if g.buffer.Len() >= minCompressBodyLenght {
		g.rw.Header().Add(VaryHeader, AcceptEncodingHeader)
	}

	if g.enabled {
		g.rw.Header().Del(ContentLengthHeader)
		g.rw.Header().Set(ContentEncodingHeader, GzipName)
	}

	g.rw.WriteHeader(g.statusCode)
}

func (g *Gzip) Flush() {
	if l := g.buffer.Len(); l < minCompressBodyLenght {
		g.enabled = false
		g.writeHeader()

		_, g.writeErr = g.write(g.buffer.Bytes())

		// Fill the buffer up to minCompressBodyLenght size.
		_, err := g.buffer.Write(make([]byte, minCompressBodyLenght-l))
		if g.writeErr == nil {
			g.writeErr = err
		}
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
	return hijack.Hijack()
}

func ModifyAcceptEncoding(header http.Header) {
	if clientSupportsGZ.MatchString(header.Get(AcceptEncodingHeader)) {
		header.Set(AcceptEncodingHeader, GzipName)
	} else {
		header.Del(AcceptEncodingHeader)
	}
}
