package eval

import "io"

type ReadCloser struct {
	io.Reader
	closer io.Closer
}

func NewReadCloser(r io.Reader, c io.Closer) *ReadCloser {
	rc := &ReadCloser{Reader: r, closer: c}
	return rc
}

func (rc ReadCloser) Close() error {
	if rc.closer == nil {
		return nil
	}
	return rc.closer.Close()
}
