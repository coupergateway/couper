package handler

import (
	"net/http"
	"os"
	"path"
)

var _ http.Handler = &Spa{}

type Spa struct {
	file string
}

func NewSpa(wd, bsFile string) *Spa {
	return &Spa{file: path.Join(wd, bsFile)}
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	file, err := os.Open(s.file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			ServeError(rw, req, http.StatusNotFound)
			return
		}

		ServeError(rw, req, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		ServeError(rw, req, http.StatusInternalServerError)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) String() string {
	return "SPA"
}
