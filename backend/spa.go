package backend

import (
	"net/http"
	"os"
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
