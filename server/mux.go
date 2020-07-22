package server

import (
	"net"
	"net/http"
	"path"
	"sort"
	"strings"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
)

type Mux struct {
	routes      map[string]routes
	fileHandler map[string]http.Handler
}

type routes []*Route

func NewMux(conf *config.Gateway) *Mux {
	routes := make(map[string]routes)
	fileHandler := make(map[string]http.Handler)
	mux := &Mux{fileHandler: fileHandler, routes: routes}

	for _, server := range conf.Server {
		var files, spa http.Handler

		if server.Spa != nil {
			spa = handler.NewSpa(server.Spa.BootstrapFile)
		}

		if server.Files != nil {
			files = handler.NewFile(server.Files.DocumentRoot, server.Files.ErrorFile)
		}

		for _, domain := range server.Domains {
			routes[domain] = make([]*Route, 0)
			if files != nil {
				fileHandler[domain] = files
			}
			if spa != nil {
				spaPath := path.Join(server.BasePath, server.Spa.BasePath)
				for _, subPath := range server.Spa.Paths {
					mux.Register(domain, path.Join(spaPath, subPath), spa)
				}
			}
		}
	}
	return mux
}

func (m *Mux) Match(req *http.Request) (http.Handler, string) {
	reqHost := stripHostPort(req.Host)
	routes, ok := m.routes[reqHost]
	if ok {
		for _, r := range routes { // routes are sorted by len desc
			if route := r.Match(req); route != nil {
				return route, r.Pattern()
			}
		}
	}

	files, fok := m.fileHandler[reqHost]
	if !fok {
		return nil, ""
	}
	return files, req.URL.Path
}

func (m *Mux) Register(domain, pattern string, handler http.Handler) {
	d := stripHostPort(domain)
	m.routes[d] = m.routes[d].append(pattern, handler)
}

func (r routes) append(pattern string, handler http.Handler) routes {
	if r == nil {
		return r
	}
	n := len(r)
	idx := sort.Search(n, func(i int) bool {
		return len(r[i].pattern) < len(pattern)
	})
	route, err := NewRoute(pattern, handler)
	if err != nil {
		panic(err)
	}
	if idx == n {
		return append(r, route)
	}

	routes := append(r, &Route{})      // try to grow the slice in place, any entry works.
	copy(routes[idx+1:], routes[idx:]) // Move shorter entries down
	routes[idx] = route
	return routes
}

// stripHostPort returns h without any trailing ":<port>".
func stripHostPort(h string) string {
	// If no port on host, return unchanged
	if strings.IndexByte(h, ':') == -1 {
		return h
	}
	host, _, err := net.SplitHostPort(h)
	if err != nil {
		return h // on error, return unchanged
	}
	return host
}
