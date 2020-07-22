package backend

import (
	"net/http"
	"os"
)

var (
	_ http.Handler = &Spa{}
	_ selectable   = &Spa{}
)

type Spa struct {
	file string
}

func NewSpa(filePath string) *Spa {
	return &Spa{file: filePath}
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	file, err := os.Open(s.file)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.NotFoundHandler().ServeHTTP(rw, req)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) hasResponse(req *http.Request) bool {
	return true
}
