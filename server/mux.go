package server

import (
	"net"
	"net/http"
	"sort"
	"strings"

	"go.avenga.cloud/couper/gateway/backend"
	"go.avenga.cloud/couper/gateway/config"
)

type Mux struct {
	routes      map[string]routes
	fileHandler map[string]http.Handler
}

type routes []*Route

func NewMux(conf *config.Gateway) *Mux {
	routes := make(map[string]routes)
	fileHandler := make(map[string]http.Handler)
	for _, server := range conf.Server {
		var files http.Handler
		if server.Files != nil {
			files = backend.NewFile(server.Files.DocumentRoot)
		}
		for _, domain := range server.Domains {
			routes[domain] = make([]*Route, 0)
			if files != nil {
				fileHandler[domain] = files
			}
		}
	}
	return &Mux{routes: routes}
}

func (m *Mux) Match(req *http.Request) (http.Handler, string) {
	reqHost := stripHostPort(req.Host)
	routes, ok := m.routes[reqHost]
	if !ok {
		files, fok := m.fileHandler[reqHost]
		if !fok {
			return nil, ""
		}
		return files, req.URL.Path
	}
	for _, r := range routes { // routes are sorted by len desc
		if route := r.Match(req); route != nil {
			return route, r.Pattern()
		}
	}
	return nil, ""
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
