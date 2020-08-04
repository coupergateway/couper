package server

import (
	"net"
	"net/http"
	"strings"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
	"go.avenga.cloud/couper/gateway/utils"
)

// Mux represents a Mux object
type Mux struct {
	api     routesMap
	apiPath map[string]string
	fs      routesMap
	fsPath  map[string]string
	spa     routesMap
	spaPath map[string]string
}

// NewMux creates a new Mux object
func NewMux(conf *config.Gateway) *Mux {
	mux := &Mux{
		api:     make(routesMap),
		apiPath: make(map[string]string),
		fs:      make(routesMap),
		fsPath:  make(map[string]string),
		spa:     make(routesMap),
		spaPath: make(map[string]string),
	}

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

			if server.API != nil {
				mux.api[domain] = make([]*Route, 0)
				mux.apiPath[domain] = server.API.BasePath

				for _, endpoint := range server.API.Endpoint {
					mux.api[domain] = mux.api[domain].add(
						utils.JoinPath(server.API.BasePath, endpoint.Pattern),
						server.API.PathHandler[endpoint],
					)
				}
			}

			if server.Files != nil {
				mux.fs[domain] = make([]*Route, 0)
				mux.fsPath[domain] = server.Files.BasePath

				mux.fs[domain] = mux.fs[domain].add(
					utils.JoinPath(server.Files.BasePath, "/**"),
					files,
				)

				// Register base_path-302 case
				if server.Files.BasePath != "/" {
					mux.fs[domain] = mux.fs[domain].add(
						strings.TrimRight(server.Files.BasePath, "/")+"$",
						files,
					)
				}
			}

			if server.Spa != nil {
				mux.spa[domain] = make([]*Route, 0)
				mux.spaPath[domain] = server.Spa.BasePath

				for _, path := range server.Spa.Paths {
					path := utils.JoinPath(server.Spa.BasePath, path)

					mux.spa[domain] = mux.spa[domain].add(
						path,
						spa,
					)

					if path != "/**" && strings.HasSuffix(path, "/**") {
						mux.spa[domain] = mux.spa[domain].add(
							path[:len(path)-len("/**")],
							spa,
						)
					}
				}
			}
		}
	}

	return mux
}

func (m *Mux) Match(req *http.Request) http.Handler {
	domain := stripHostPort(req.Host)

	if m.api != nil {
		if h, ok := m.api.Match(domain, req); ok {
			return h
		}

		if _, ok := m.api[domain]; ok && m.isAPIError(req.URL.Path, domain) {
			// TODO: RETURN API-ERROR (JSON)
		}
	}
	if m.fs != nil {
		if h, ok := m.fs.Match(domain, req); ok {
			if a, ok := h.(handler.Lookupable); ok && a.HasResponse(req) {
				return h
			}
		}
	}
	if m.spa != nil {
		if h, ok := m.spa.Match(domain, req); ok {
			return h
		}
	}

	if m.fs != nil && m.isFSError(req.URL.Path, domain) {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			handler.ServeError(rw, r, http.StatusNotFound)
		})
	}

	return nil
}

func (m *Mux) isAPIError(reqPath, domain string) bool {
	p1 := m.apiPath[domain]
	p2 := m.apiPath[domain]

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		if m.fs != nil && m.apiPath[domain] == m.fsPath[domain] {
			return false
		}
		if m.spa != nil && m.apiPath[domain] == m.spaPath[domain] {
			return false
		}

		return true
	}

	return false
}

func (m *Mux) isFSError(reqPath, domain string) bool {
	p1 := m.fsPath[domain]
	p2 := m.fsPath[domain]

	if p2 != "/" {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		return true
	}

	return false
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
