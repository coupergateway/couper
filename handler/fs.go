package handler

import "net/http"

var _ http.Handler = &FS{}

type FS struct {
	status int
}

func NewFS(status int) *FS {
	return &FS{status: status}
}

func (f *FS) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ServeError(rw, req, f.status)
}

func (f *FS) String() string {
	return "File"
}
