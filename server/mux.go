package server

import (
	"net"
	"net/http"
	"sort"
	"strings"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
	"go.avenga.cloud/couper/gateway/utils"
)

type Mux struct {
	routes map[string]routes
}

type routes []*Route

func NewMux(conf *config.Gateway) *Mux {
	mux := &Mux{routes: make(map[string]routes)}

	for _, server := range conf.Server {
		var files, spa http.Handler

		if server.Files != nil {
			files = handler.NewFile(conf.WD, server.Files.BasePath, server.Files.DocumentRoot, server.Files.ErrorFile)
		}

		if server.Spa != nil {
			spa = handler.NewSpa(conf.WD, server.Spa.BootstrapFile)
		}

		for _, domain := range server.Domains {
			domain := stripHostPort(domain)

			mux.routes[domain] = make([]*Route, 0)

			if server.Files != nil {
				mux.register(domain, utils.JoinPath(server.Files.BasePath, "/**"), files)

				// Register base_path-302 case
				if server.Files.BasePath != "/" {
					base := strings.TrimRight(server.Files.BasePath, "/") + "$"
					mux.register(domain, base, files)
				}
			}

			if server.API != nil {
				for _, endpoint := range server.API.Endpoint {
					pattern := utils.JoinPath(server.API.BasePath, endpoint.Pattern)

					// TODO: shadow clone slice per domain (len(server.Domains) > 1)
					for _, domain := range server.Domains {
						mux.register(domain, pattern, server.API.PathHandler[endpoint])
					}
				}
			}

			if server.Spa != nil {
				for _, path := range server.Spa.Paths {
					mux.register(domain, utils.JoinPath(server.Spa.BasePath, path), spa)
				}
			}
		}
	}

	return mux
}

func (m *Mux) Match(req *http.Request) (http.Handler, string) {
	reqHost := stripHostPort(req.Host)

	if routes, ok := m.routes[reqHost]; ok {
		for _, r := range routes { // routes are sorted by len desc
			if route := r.Match(req); route != nil {
				return route, r.Pattern()
			}
		}
	}

	return nil, ""
}

func (m *Mux) register(domain, pattern string, handler http.Handler) {
	m.routes[domain] = m.routes[domain].append(pattern, handler)
}

func (r routes) append(pattern string, h http.Handler) routes {
	route, err := NewRoute(pattern, h)
	if err != nil {
		panic(err)
	}

	for n, v := range r {
		if v.pattern == route.pattern {
			route, err = NewRoute(pattern, handler.NewSelector(v.handler, h))
			if err != nil {
				panic(err)
			}

			r[n] = route
			return r
		}
	}

	n := len(r)
	idx := sort.Search(n, func(i int) bool {
		return (r[i].sortLen) < (route.sortLen)
	})
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
