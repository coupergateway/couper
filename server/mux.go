package server

import (
	"net"
	"net/http"
	"path"
	"strings"

	"go.avenga.cloud/couper/gateway/errors"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
	"go.avenga.cloud/couper/gateway/utils"
)

// Mux represents a Mux object
type Mux struct {
	api       routesMap
	apiPath   map[string]string
	apiErrTpl *errors.Template
	fs        routesMap
	fsPath    map[string]string
	fsErrTpl  *errors.Template
	spa       routesMap
	spaPath   map[string]string
}

// NewMux creates a new Mux object
func NewMux(conf *config.Gateway, ph pathHandler) *Mux {
	mux := &Mux{
		api:     make(routesMap),
		apiPath: make(map[string]string),
		fs:      make(routesMap),
		fsPath:  make(map[string]string),
		spa:     make(routesMap),
		spaPath: make(map[string]string),
	}

	var err error

	for _, server := range conf.Server {
		var files, spa http.Handler

		if server.Files != nil {
			if server.Files.ErrorFile != "" {
				mux.fsErrTpl, err = errors.NewTemplateFromFile(path.Join(conf.WorkDir, server.Files.ErrorFile))
				if err != nil {
					panic(err)
				}
			}
			if mux.fsErrTpl == nil {
				mux.fsErrTpl = errors.DefaultHTML
			}
			files = handler.NewFile(conf.WorkDir, server.Files.BasePath, server.Files.DocumentRoot, mux.fsErrTpl)
		}

		if server.Spa != nil {
			spa = handler.NewSpa(conf.WorkDir, server.Spa.BootstrapFile)
		}

		if server.API != nil && server.API.ErrorFile != "" {
			mux.apiErrTpl, err = errors.NewTemplateFromFile(path.Join(conf.WorkDir, server.API.ErrorFile))
			if err != nil {
				panic(err)
			}
		}
		if mux.apiErrTpl == nil {
			mux.apiErrTpl = errors.DefaultJSON
		}

		for _, domain := range server.Domains {
			domain := stripHostPort(domain)

			if server.API != nil {
				mux.api[domain] = make([]*Route, 0)
				mux.apiPath[domain] = server.API.BasePath

				for _, endpoint := range server.API.Endpoint {
					mux.api[domain] = mux.api[domain].add(
						utils.JoinPath(server.API.BasePath, endpoint.Pattern),
						ph[endpoint],
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

				for _, spaPath := range server.Spa.Paths {
					spaPath := utils.JoinPath(server.Spa.BasePath, spaPath)

					mux.spa[domain] = mux.spa[domain].add(
						spaPath,
						spa,
					)

					if spaPath != "/**" && strings.HasSuffix(spaPath, "/**") {
						mux.spa[domain] = mux.spa[domain].add(
							spaPath[:len(spaPath)-len("/**")],
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

	if len(m.api) > 0 {
		if h, ok := m.api.Match(domain, req); ok {
			return h
		}

		if m.isAPIError(req.URL.Path, domain) {
			return m.apiErrTpl.ServeError(errors.APIRouteNotFound)
		}
	}

	if len(m.fs) > 0 {
		if h, ok := m.fs.Match(domain, req); ok {
			if a, ok := h.(handler.Lookupable); ok && a.HasResponse(req) {
				return h
			}
		}
	}

	if len(m.spa) > 0 {
		if h, ok := m.spa.Match(domain, req); ok {
			return h
		}
	}

	if len(m.fs) > 0 && m.isFileError(req.URL.Path, domain) {
		return m.fsErrTpl.ServeError(errors.FilesRouteNotFound)
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
		if len(m.fs) > 0 && m.apiPath[domain] == m.fsPath[domain] {
			return false
		}
		if len(m.spa) > 0 && m.apiPath[domain] == m.spaPath[domain] {
			return false
		}

		return true
	}

	return false
}

func (m *Mux) isFileError(reqPath, domain string) bool {
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
