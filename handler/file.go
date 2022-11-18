package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/server/writer"
	"github.com/avenga/couper/utils"
)

const dirIndexFile = "index.html"

var (
	_ http.Handler   = &File{}
	_ server.Context = &File{}
)

type File struct {
	basePath   string
	errorTpl   *errors.Template
	modifier   []hcl.Body
	preferSPA  PreferSPAfn
	rootDir    http.Dir
	srvOptions *server.Options
}

func NewFile(docRoot, basePath string, preferFn PreferSPAfn, errorTpl *errors.Template, srvOpts *server.Options, modifier []hcl.Body) (*File, error) {
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
		basePath:   basePath,
		errorTpl:   errorTpl,
		modifier:   modifier,
		preferSPA:  preferFn,
		srvOptions: srvOpts,
		rootDir:    http.Dir(dir),
	}

	return f, nil
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		f.errorTpl.WithError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	if info.IsDir() {
		f.serveDirectory(reqPath, rw, req)
		return
	}

	if r, ok := rw.(*writer.Response); ok {
		evalContext := eval.ContextFromRequest(req)
		r.AddModifier(evalContext.HCLContext(), f.modifier...)
	}

	http.ServeContent(rw, req, reqPath, info.ModTime(), file)
}

func (f *File) serveDirectory(reqPath string, rw http.ResponseWriter, req *http.Request) {
	if !f.HasResponse(req) {
		f.errorTpl.WithError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}

	if !strings.HasSuffix(reqPath, "/") {
		if r, ok := rw.(*writer.Response); ok {
			evalContext := eval.ContextFromRequest(req)
			r.AddModifier(evalContext.HCLContext(), f.modifier...)
		}

		rw.Header().Set("Location", utils.JoinPath(req.URL.Path, "/"))
		rw.WriteHeader(http.StatusFound)
		return
	}

	reqPath = path.Join(reqPath, dirIndexFile)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil || info.IsDir() {
		f.errorTpl.WithError(errors.RouteNotFound).ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	if r, ok := rw.(*writer.Response); ok {
		evalContext := eval.ContextFromRequest(req)
		r.AddModifier(evalContext.HCLContext(), f.modifier...)
	}

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
		filePath := path.Join(reqPath, "/", dirIndexFile)

		file, info, err = f.openDocRootFile(filePath)
		if err != nil {
			return false
		}
		defer file.Close()

		if info.IsDir() {
			return false
		}

		return !f.preferSPA(reqPath)
	}

	// TODO improve performance for this range call
	return !f.preferSPA(reqPath)
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

func (f *File) Options() *server.Options {
	return f.srvOptions
}

func (f *File) String() string {
	return "file"
}

type PreferSPAfn func(string) bool

func NewPreferSpaFn(bootstrapFiles []string, docRoot string) PreferSPAfn {
	absDocRoot, _ := filepath.Abs(docRoot)
	files := bootstrapFiles[:]
	return func(subPath string) bool {
		fileHandlerPath := filepath.Join(absDocRoot, subPath)
		for _, f := range files {
			if filepath.Dir(f) == fileHandlerPath || // baseDir index case
				f == fileHandlerPath { // direct spa file call
				return true
			}
		}
		return false
	}
}
