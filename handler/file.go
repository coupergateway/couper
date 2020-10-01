package handler

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/avenga/couper/errors"

	"github.com/avenga/couper/utils"
)

const dirIndexFile = "index.html"

var (
	_ http.Handler = &File{}
	_ HasResponse  = &File{}
)

type HasResponse interface {
	HasResponse(req *http.Request) bool
}

type File struct {
	basePath string
	errorTpl *errors.Template
	rootDir  http.Dir
}

func NewFile(basePath, docRoot string, errTpl *errors.Template) *File {
	f := &File{
		basePath: basePath,
		errorTpl: errTpl,
		rootDir:  http.Dir(docRoot),
	}

	return f
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		f.errorTpl.ServeError(errors.FilesRouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	if info.IsDir() {
		f.serveDirectory(reqPath, rw, req)
		return
	}

	// TODO: gzip, br?
	http.ServeContent(rw, req, reqPath, info.ModTime(), file)
}

func (f *File) serveDirectory(reqPath string, rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(reqPath, "/") {
		rw.Header().Set("Location", utils.JoinPath(req.URL.Path, "/"))
		rw.WriteHeader(http.StatusFound)
		return
	}

	reqPath = path.Join(reqPath, dirIndexFile)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil || info.IsDir() {
		f.errorTpl.ServeError(errors.FilesRouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	// TODO: gzip, br?
	http.ServeContent(rw, req, reqPath, info.ModTime(), file)
}

func (f *File) HasResponse(req *http.Request) bool {
	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		return false
	}
	file.Close()

	if info.IsDir() {
		reqPath = path.Join(reqPath, "/", dirIndexFile)

		file, info, err := f.openDocRootFile(reqPath)
		if err != nil {
			return false
		}
		defer file.Close()

		if info.IsDir() {
			return false
		}
	}

	return true
}

func (f *File) openDocRootFile(name string) (http.File, os.FileInfo, error) {
	file, err := f.rootDir.Open(name)
	if err != nil {
		return nil, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return file, info, nil
}

func (f *File) removeBasePath(reqPath string) string {
	if strings.HasPrefix(reqPath, f.basePath) {
		return utils.JoinPath("/", reqPath[len(f.basePath):])
	} else if f.basePath != "/" {
		base := strings.TrimRight(f.basePath, "/")

		if strings.HasPrefix(reqPath, base) {
			return utils.JoinPath("/", reqPath[len(base):])
		}
	}

	return reqPath
}

func (f *File) Template() *errors.Template {
	return f.errorTpl
}

func (f *File) String() string {
	return "file"
}
