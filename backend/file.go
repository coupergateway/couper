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
	if status != http.StatusNotFound {
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

func wrapHandler(h http.Handler, bf string, spa_paths []string) (http.HandlerFunc, error) {
	dir, _ := os.Getwd()
	bs_content, err := ioutil.ReadFile(path.Join(dir, bf))
	if err != nil {
		return nil, err
	}
	return func(rw http.ResponseWriter, req *http.Request) {
		sh := &SpaHandler{rw: rw}
		h.ServeHTTP(sh, req)
		if sh.status == http.StatusNotFound {
			for _, a := range spa_paths {
				// TODO implement wildcard match
				if req.URL.Path == a {
					rw.Write(bs_content)
					return
				}
			}
			rw.Write(sh.body)
		}
	}, nil
}

func NewFile(root string, log *logrus.Entry, bf string, spa_paths []string) (*File, error) {
	dir, _ := os.Getwd()
	wrapper, err := wrapHandler(http.FileServer(http.Dir(path.Join(dir, root))), bf, spa_paths)
	if err != nil {
		return nil, err
	}
	return &File{
		log:     log,
		handler: wrapper,
	}, nil
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	sh := &SpaHandler{rw: rw}
	f.handler.ServeHTTP(sh, req)
}
