package backend

import (
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

func NewFile(root string, log *logrus.Entry) *File {
	dir, _ := os.Getwd()
	return &File{
		log:     log,
		handler: http.FileServer(http.Dir(path.Join(dir, root))),
	}
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f.handler.ServeHTTP(rw, req)
}
