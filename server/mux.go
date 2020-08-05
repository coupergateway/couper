package server

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.avenga.cloud/couper/gateway/assets"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/handler"
	"go.avenga.cloud/couper/gateway/utils"
)

// Mux represents a Mux object
type Mux struct {
	api     routesMap
	apiPath map[string]string
	apiErr  map[string]*assets.AssetFile
	fs      routesMap
	fsPath  map[string]string
	fsErr   map[string]*assets.AssetFile
	spa     routesMap
	spaPath map[string]string
}

// NewMux creates a new Mux object
func NewMux(conf *config.Gateway, ph pathHandler) *Mux {
	mux := &Mux{
		api:     make(routesMap),
		apiErr:  make(map[string]*assets.AssetFile),
		apiPath: make(map[string]string),
		fs:      make(routesMap),
		fsErr:   make(map[string]*assets.AssetFile),
		fsPath:  make(map[string]string),
		spa:     make(routesMap),
		spaPath: make(map[string]string),
	}

	for _, server := range conf.Server {
		var files, spa http.Handler

		apiErrAsset, fsErrAsset := getErrorAssets(conf.WorkDir, server)

		if server.Files != nil {
			files = handler.NewFile(conf.WorkDir, server.Files.BasePath, server.Files.DocumentRoot, fsErrAsset)
		}

		if server.Spa != nil {
			spa = handler.NewSpa(conf.WorkDir, server.Spa.BootstrapFile)
		}

		for _, domain := range server.Domains {
			domain := stripHostPort(domain)

			if server.API != nil {
				mux.api[domain] = make([]*Route, 0)
				mux.apiPath[domain] = server.API.BasePath
				mux.apiErr[domain] = apiErrAsset

				for _, endpoint := range server.API.Endpoint {
					h := ph[endpoint]
					if v, ok := h.(handler.ErrorHandleable); ok {
						v.SetErrorAsset(apiErrAsset)
					}

					mux.api[domain] = mux.api[domain].add(
						utils.JoinPath(server.API.BasePath, endpoint.Pattern),
						ph[endpoint],
					)
				}
			}

			if server.Files != nil {
				mux.fs[domain] = make([]*Route, 0)
				mux.fsPath[domain] = server.Files.BasePath
				mux.fsErr[domain] = fsErrAsset

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

	if len(m.api) > 0 {
		if h, ok := m.api.Match(domain, req); ok {
			return h
		}

		if m.isAPIError(req.URL.Path, domain) {
			return handler.NewErrorHandler(m.apiErr[domain], 4001, http.StatusNotFound)
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

	if len(m.fs) > 0 && m.isFSError(req.URL.Path, domain) {
		return handler.NewErrorHandler(m.fsErr[domain], 3001, http.StatusNotFound)
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

func getErrorAssets(wd string, server *config.Server) (*assets.AssetFile, *assets.AssetFile) {
	var apiErrAsset, fsErrAsset *assets.AssetFile

	if server.API != nil && server.API.ErrorFile != "" {
		file, info, err := openFile(path.Join(wd, server.API.ErrorFile))
		if err != nil {
			panic(err)
		}
		file.Close()

		body, _ := ioutil.ReadFile(path.Join(wd, server.API.ErrorFile))
		ct := mime.TypeByExtension(filepath.Ext(server.API.ErrorFile))
		size := fmt.Sprintf("%d", info.Size())
		apiErrAsset = assets.NewAssetFile(body, ct, size)
	} else {
		a, err := assets.Assets.Open("error.json")
		if err != nil {
			panic(err)
		}
		apiErrAsset = a
	}

	if server.Files != nil && server.Files.ErrorFile != "" {
		file, info, err := openFile(path.Join(wd, server.Files.ErrorFile))
		if err != nil {
			panic(err)
		}
		file.Close()

		body, _ := ioutil.ReadFile(path.Join(wd, server.Files.ErrorFile))
		ct := mime.TypeByExtension(filepath.Ext(server.Files.ErrorFile))
		size := fmt.Sprintf("%d", info.Size())
		fsErrAsset = assets.NewAssetFile(body, ct, size)
	} else {
		a, err := assets.Assets.Open("error.html")
		if err != nil {
			panic(err)
		}
		fsErrAsset = a
	}

	apiErrAsset.MakeTemplate()
	fsErrAsset.MakeTemplate()

	return apiErrAsset, fsErrAsset
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
