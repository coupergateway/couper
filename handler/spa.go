package handler

import (
	"net/http"
	"os"
	"path"
)

var (
	_ http.Handler = &Spa{}
	_ selectable   = &Spa{}
)

type Spa struct {
	file string
}

func NewSpa(wd, bsFile string) *Spa {
	return &Spa{file: path.Join(wd, bsFile)}
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	file, err := os.Open(s.file)
	if err != nil {
		ServeError(rw, req, http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		ServeError(rw, req, http.StatusNotFound)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) hasResponse(req *http.Request) bool {
	return true
}

func (s *Spa) String() string {
	return "SPA"
}
