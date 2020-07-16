package backend

import (
	"net/http"
	"os"
	"path"

	"github.com/hashicorp/hcl/v2"
)

var _ http.Handler = &File{}

type File struct {
	ContextOptions hcl.Body `hcl:",remain"`
	handler        http.Handler
}

func NewFile(root string) *File {
	dir, _ := os.Getwd()
	return &File{
		handler: http.FileServer(http.Dir(path.Join(dir, root))),
	}
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f.handler.ServeHTTP(rw, req)
}
