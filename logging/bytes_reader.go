package logging

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/coupergateway/couper/config/request"
)

var _ io.ReadCloser = &BytesCountReader{}

type BytesCountReader struct {
	c context.Context
	n int64
	r io.ReadCloser
}

// NewBytesCountReader just counts the raw read bytes from given response body for logging purposes.
func NewBytesCountReader(beresp *http.Response) io.ReadCloser {
	return &BytesCountReader{
		c: beresp.Request.Context(),
		r: beresp.Body,
	}
}

func (b *BytesCountReader) Read(p []byte) (n int, err error) {
	n, err = b.r.Read(p)
	b.n += int64(n)
	return n, err
}

func (b *BytesCountReader) Close() error {
	bytesPtr, ok := b.c.Value(request.BackendBytes).(*int64)
	if ok {
		atomic.StoreInt64(bytesPtr, b.n)
	}
	return b.r.Close()
}
