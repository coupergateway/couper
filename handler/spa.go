package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/avenga/couper/errors"
)

var _ http.Handler = &Spa{}

type Spa struct {
	file string
}

func NewSpa(bootstrapFile string) (*Spa, error) {
	absPath, err := filepath.Abs(bootstrapFile)
	if err != nil {
		return nil, err
	}
	return &Spa{file: absPath}, nil
}

func (s *Spa) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, err := os.Open(s.file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			errors.DefaultHTML.ServeError(errors.SPARouteNotFound).ServeHTTP(rw, req)
			return
		}

		errors.DefaultHTML.ServeError(errors.SPAError).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		errors.DefaultHTML.ServeError(errors.SPAError).ServeHTTP(rw, req)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) String() string {
	return "spa"
}
