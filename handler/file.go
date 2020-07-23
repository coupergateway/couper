package handler

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"go.avenga.cloud/couper/gateway/utils"
)

const dirIndexFile = "index.html"

var (
	_ http.Handler = &File{}
	_ selectable   = &File{}
)

type File struct {
	basePath string
	errFile  string
	rootDir  http.Dir
}

func NewFile(wd, basePath, docRoot, errFile string) *File {
	f := &File{
		basePath: basePath,
		rootDir:  http.Dir(path.Join(wd, docRoot)),
	}

	if errFile != "" {
		f.errFile = path.Join(wd, errFile)
	}

	return f
}

func (f *File) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		f.serveErrFile(rw, req)
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

func (f *File) serveErrFile(rw http.ResponseWriter, req *http.Request) {
	file, info, err := openFile(f.errFile)
	if f.errFile == "" || err != nil {
		ServeError(rw, req, http.StatusNotFound)
		return
	}
	defer file.Close()

	ct := mime.TypeByExtension(filepath.Ext(f.errFile))
	if ct != "" {
		rw.Header().Set("Content-Type", ct)
	}

	rw.WriteHeader(http.StatusNotFound)

	// TODO: gzip, br?
	if req.Method != "HEAD" {
		rw.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		rw.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		io.Copy(rw, file) // TODO: log
	}
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
		f.serveErrFile(rw, req)
		return
	}
	defer file.Close()

	// TODO: gzip, br?
	http.ServeContent(rw, req, reqPath, info.ModTime(), file)
}

func (f *File) hasResponse(req *http.Request) bool {
	reqPath := f.removeBasePath(req.URL.Path)

	file, info, err := f.openDocRootFile(reqPath)
	if err != nil {
		return false
	}
	defer file.Close()

	if info.IsDir() {
		reqPath := path.Join(reqPath, dirIndexFile)

		index, info, err := f.openDocRootFile(reqPath)
		if err != nil {
			return false
		}
		defer index.Close()

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

func (f *File) String() string {
	return "File"
}
