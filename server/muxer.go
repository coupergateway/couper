package server

import (
	"net/http"
	"strings"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
)

// Muxer represents the Muxer object.
type Muxer struct {
	mux *runtime.Mux
}

// NewMuxer creates a new Muxer object
func NewMuxer(mux *runtime.Mux) *Muxer {
	return &Muxer{mux: mux}
}

// Match tries to find a http.Handler by the given request
func (m *Muxer) Match(req *http.Request) http.Handler {
	if len(m.mux.API) > 0 {
		if h, ok := NewRouter(m.mux.API).Match(req); ok {
			return h
		}

		if m.isAPIError(req.URL.Path) {
			return m.mux.APIErrTpl.ServeError(errors.APIRouteNotFound)
		}
	}

	if len(m.mux.FS) > 0 {
		if h, ok := NewRouter(m.mux.FS).Match(req); ok {
			fileHandler := h
			if p, isProtected := h.(ac.ProtectedHandler); isProtected {
				fileHandler = p.Child()
			}
			if fh, ok := fileHandler.(handler.HasResponse); ok && fh.HasResponse(req) {
				return h
			}
		}
	}

	if len(m.mux.SPA) > 0 {
		if h, ok := NewRouter(m.mux.SPA).Match(req); ok {
			return h
		}
	}

	if len(m.mux.FS) > 0 && m.isFileError(req.URL.Path) {
		return m.mux.FSErrTpl.ServeError(errors.FilesRouteNotFound)
	}

	return nil
}

func (m *Muxer) isAPIError(reqPath string) bool {
	p1 := m.mux.APIPath
	p2 := m.mux.APIPath

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		if len(m.mux.FS) > 0 && m.mux.APIPath == m.mux.FSPath {
			return false
		}
		if len(m.mux.SPA) > 0 && m.mux.APIPath == m.mux.SPAPath {
			return false
		}

		return true
	}

	return false
}

func (m *Muxer) isFileError(reqPath string) bool {
	p1 := m.mux.FSPath
	p2 := m.mux.FSPath

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		return true
	}

	return false
}
