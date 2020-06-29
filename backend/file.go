package backend

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
)

var _ http.Handler = &File{}

type File struct {
	ContextOptions hcl.Body `hcl:",remain"`
	log            *logrus.Entry
	handler        http.Handler
}

type SpaHandler struct {
	rw     http.ResponseWriter
	status int
	body   []byte
}

func (sh *SpaHandler) Header() http.Header {
	return sh.rw.Header()
}

func (sh *SpaHandler) WriteHeader(status int) {
	sh.status = status
	if (status != http.StatusNotFound) {
		sh.rw.WriteHeader(status)
	}
}

func (sh *SpaHandler) Write(p []byte) (int, error) {
	if sh.status != http.StatusNotFound {
		return sh.rw.Write(p)
	}
	sh.body = p
	return len(p), nil
}

func wrapHandler(h http.Handler, bf string, spa_paths []string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		sh := &SpaHandler{rw: rw}
		h.ServeHTTP(sh, req)
		if sh.status == http.StatusNotFound {
			for _, a := range spa_paths {
				// TODO implement wildcard match
				if req.URL.Path == a {
					dir, _ := os.Getwd()
					bs_content, _ := ioutil.ReadFile(path.Join(dir, bf))
					rw.Write(bs_content)
					return
				}
			}
			rw.Write(sh.body)
		}
	}
}

func NewFile(root string, log *logrus.Entry, bf string, spa_paths []string) *File {
	dir, _ := os.Getwd()
	return &File{
		log:     log,
		handler: wrapHandler(http.FileServer(http.Dir(path.Join(dir, root))), bf, spa_paths),
	}
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	sh := &SpaHandler{rw: rw}
	f.handler.ServeHTTP(sh, req)
}
