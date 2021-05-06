package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/utils"
)

const dirIndexFile = "index.html"

var (
	_ HasResponse    = &File{}
	_ http.Handler   = &File{}
	_ server.Context = &File{}
)

type HasResponse interface {
	HasResponse(req *http.Request) bool
}

type File struct {
	basePath   string
	rootDir    http.Dir
	srvOptions *server.Options
}

func NewFile(docRoot string, srvOpts *server.Options) (*File, error) {
	dir, err := filepath.Abs(docRoot)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("document root must be a directory: %q", docRoot)
	}

	f := &File{
		basePath:   srvOpts.FilesBasePath,
		srvOptions: srvOpts,
		rootDir:    http.Dir(dir),
	}

	return f, nil
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		f.srvOptions.FilesErrTpl.ServeError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	if info.IsDir() {
		f.serveDirectory(reqPath, rw, req)
		return
	}

	http.ServeContent(rw, req, reqPath, info.ModTime(), file)
}

func (f *File) serveDirectory(reqPath string, rw http.ResponseWriter, req *http.Request) {
	if !f.HasResponse(req) {
		f.srvOptions.FilesErrTpl.ServeError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}

	if !strings.HasSuffix(reqPath, "/") {
		rw.Header().Set("Location", utils.JoinPath(req.URL.Path, "/"))
		rw.WriteHeader(http.StatusFound)
		return
	}

	reqPath = path.Join(reqPath, dirIndexFile)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil || info.IsDir() {
		f.srvOptions.FilesErrTpl.ServeError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

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

		file, info, err = f.openDocRootFile(reqPath)
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
	return f.srvOptions.FilesErrTpl
}

func (f *File) Options() *server.Options {
	return f.srvOptions
}

func (f *File) String() string {
	return "file"
}
