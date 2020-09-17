package server

import (
	"context"
	"net/http"
	"strings"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
)

// Muxer represents the Muxer object.
type Muxer struct{}

// NewMuxer creates a new Muxer object
func NewMuxer() *Muxer {
	return &Muxer{}
}

// Match tries to find a http.Handler by the given request
func (m *Muxer) Match(mux *runtime.Mux, req *http.Request) http.Handler {
	if m == nil {
		return nil
	}

	if len(mux.API) > 0 {
		if h, ok := m.matchRoute(mux.API, req); ok {
			return h
		}

		if m.isAPIError(mux, req.URL.Path) {
			return mux.APIErrTpl.ServeError(errors.APIRouteNotFound)
		}
	}

	if len(mux.FS) > 0 {
		if h, ok := m.matchRoute(mux.FS, req); ok {
			fileHandler := h
			if p, isProtected := h.(ac.ProtectedHandler); isProtected {
				fileHandler = p.Child()
			}
			if fh, ok := fileHandler.(handler.HasResponse); ok && fh.HasResponse(req) {
				return h
			}
		}
	}

	if len(mux.SPA) > 0 {
		if h, ok := m.matchRoute(mux.SPA, req); ok {
			return h
		}
	}

	if len(mux.FS) > 0 && m.isFileError(mux, req.URL.Path) {
		return mux.FSErrTpl.ServeError(errors.FilesRouteNotFound)
	}

	return nil
}

func (m *Muxer) isAPIError(mux *runtime.Mux, reqPath string) bool {
	p1 := mux.APIPath
	p2 := mux.APIPath

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		if len(mux.FS) > 0 && mux.APIPath == mux.FSPath {
			return false
		}
		if len(mux.SPA) > 0 && mux.APIPath == mux.SPAPath {
			return false
		}

		return true
	}

	return false
}

func (m *Muxer) isFileError(mux *runtime.Mux, reqPath string) bool {
	p1 := mux.FSPath
	p2 := mux.FSPath

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		return true
	}

	return false
}

func (m *Muxer) matchRoute(routes runtime.Routes, req *http.Request) (http.Handler, bool) {
	var wildcardRoutes runtime.Routes

	if len(routes) == 0 {
		return nil, false
	}

	for _, route := range routes {
		if route.HasWildcard() {
			wildcardRoutes = append(wildcardRoutes, route)
			continue
		}
		if h := m.matchHandler(route, req); h != nil {
			return h, true
		}
	}

	for _, route := range wildcardRoutes {
		if h := m.matchHandler(route, req); h != nil {
			return h, true
		}
	}

	return nil, false
}

func (m *Muxer) matchHandler(route *runtime.Route, req *http.Request) http.Handler {
	if route.GetMatcher().MatchString(req.URL.Path) {
		if route.HasWildcard() {
			match := route.GetMatcher().FindStringSubmatch(req.URL.Path)
			if len(match) > 1 {
				*req = *req.WithContext(context.WithValue(req.Context(), request.Wildcard, match[2]))
			}
		}
		*req = *req.WithContext(context.WithValue(req.Context(), request.Endpoint, route.Name()))
		return route.GetHandler()
	}
	return nil
}
