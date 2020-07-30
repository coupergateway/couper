package handler

import (
	"net/http"
	"os"
	"path"

	"go.avenga.cloud/couper/gateway/assets"
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
			asset, _ := assets.Assets.Open("error.html")
			asset.MakeTemplate()
			NewErrorHandler(asset, 2001, http.StatusNotFound).ServeHTTP(rw, req)
			return
		}

		asset, _ := assets.Assets.Open("error.html")
		NewErrorHandler(asset, 2001, http.StatusInternalServerError).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil || fileInfo.IsDir() {
		asset, _ := assets.Assets.Open("error.html")
		asset.MakeTemplate()
		NewErrorHandler(asset, 2001, http.StatusInternalServerError).ServeHTTP(rw, req)
		return
	}

	http.ServeContent(rw, req, s.file, fileInfo.ModTime(), file)
}

func (s *Spa) String() string {
	return "SPA"
}
