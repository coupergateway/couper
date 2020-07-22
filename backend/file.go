package backend

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	_ http.Handler = &File{}
	_ selectable   = &File{}
)

type File struct {
	errFile string
	rootDir http.Dir
}

func NewFile(wd, docRoot, errFile string) *File {
	return &File{
		errFile: path.Join(wd, errFile),
		rootDir: http.Dir(path.Join(wd, docRoot)),
	}
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	urlPath := req.URL.Path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
		req.URL.Path = urlPath
	}
	urlPath = path.Clean(urlPath)
	file, err := f.rootDir.Open(urlPath)
	if err != nil {
		f.serveErrFile(rw, req)
		return
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	// no directory listing
	if err != nil || fileInfo.IsDir() {
		f.serveErrFile(rw, req)
		return
	}

	http.ServeContent(rw, req, urlPath, fileInfo.ModTime(), file)
}

func (f *File) serveErrFile(rw http.ResponseWriter, req *http.Request) {
	if f.errFile == "" {
		http.NotFoundHandler().ServeHTTP(rw, req)
		return
	}

	file, info, err := openFile(f.errFile)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(rw, req)
		return
	}
	defer file.Close()

	ct := mime.TypeByExtension(filepath.Ext(f.errFile))
	if ct != "" {
		rw.Header().Set("Content-Type", ct)
	}

	rw.WriteHeader(http.StatusNotFound)

	// TODO: gzip?
	if req.Method != "HEAD" {
		rw.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		rw.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		io.Copy(rw, file) // TODO: log
	}
}

func (f *File) hasResponse(req *http.Request) bool {
	return true
}

func openFile(name string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(name)
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
