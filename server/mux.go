package server

import (
	"context"
	"net/http"
	"sort"
	"strings"

	gmux "github.com/gorilla/mux"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/utils"
)

// Mux is a http request router and dispatches requests
// to their corresponding http handlers.
type Mux struct {
	endpointRoot *gmux.Router
	fileRoot     *gmux.Router
	opts         *runtime.MuxOptions
	spaRoot      *gmux.Router
}

const (
	serverOptionsKey = "serverContextOptions"
	wildcardSearch   = "/**"
)

func isParamSegment(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func SortPathPatterns(pathPatterns []string) {
	sort.Slice(pathPatterns, func(i, j int) bool {
		iSegments := strings.Split(strings.TrimPrefix(pathPatterns[i], "/"), "/")
		jSegments := strings.Split(strings.TrimPrefix(pathPatterns[j], "/"), "/")
		iLastSegment := iSegments[len(iSegments)-1]
		jLastSegment := jSegments[len(jSegments)-1]
		if iLastSegment != "**" && jLastSegment == "**" {
			return true
		}
		if iLastSegment == "**" && jLastSegment != "**" {
			return false
		}
		if len(iSegments) > len(jSegments) {
			return true
		}
		if len(iSegments) < len(jSegments) {
			return false
		}
		for k, iSegment := range iSegments {
			jSegment := jSegments[k]
			if !isParamSegment(iSegment) && isParamSegment(jSegment) {
				return true
			}
			if isParamSegment(iSegment) && !isParamSegment(jSegment) {
				return false
			}
		}
		return sort.StringSlice{pathPatterns[i], pathPatterns[j]}.Less(0, 1)
	})
}

func sortedPathPatterns(routes map[string]http.Handler) []string {
	pathPatterns := make([]string, len(routes))
	i := 0
	for k, _ := range routes {
		pathPatterns[i] = k
		i++
	}
	SortPathPatterns(pathPatterns)
	return pathPatterns
}

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = runtime.NewMuxOptions()
	}

	mux := &Mux{
		opts:         opts,
		endpointRoot: gmux.NewRouter(),
		fileRoot:     gmux.NewRouter(),
		spaRoot:      gmux.NewRouter(),
	}

	return mux
}

func (m *Mux) RegisterConfigured() {
	for _, path := range sortedPathPatterns(m.opts.EndpointRoutes) {
		// TODO: handle method option per endpoint configuration
		mustAddRoute(m.endpointRoot, path, m.opts.EndpointRoutes[path], true)
	}

	for _, path := range sortedPathPatterns(m.opts.FileRoutes) {
		mustAddRoute(m.fileRoot, utils.JoinOpenAPIPath(path, "/**"), m.opts.FileRoutes[path], false)
	}

	for _, path := range sortedPathPatterns(m.opts.SPARoutes) {
		mustAddRoute(m.spaRoot, path, m.opts.SPARoutes[path], true)
	}
}

var noDefaultMethods []string

func registerHandler(root *gmux.Router, methods []string, path string, handler http.Handler) {
	notAllowedMethodsHandler := errors.DefaultJSON.WithError(errors.MethodNotAllowed)
	allowedMethodsHandler := middleware.NewAllowedMethodsHandler(methods, noDefaultMethods, handler, notAllowedMethodsHandler)
	mustAddRoute(root, path, allowedMethodsHandler, false)
}

func (m *Mux) FindHandler(req *http.Request) http.Handler {
	ctx := context.WithValue(req.Context(), request.ServerName, m.opts.ServerOptions.ServerName)
	routeMatch, matches := m.match(m.endpointRoot, req)
	if !matches {
		// No matches for api or free endpoints. Determine if we have entered an api basePath
		// and handle api related errors accordingly.
		// Otherwise, look for existing files or spa fallback.
		if tpl, api := m.getAPIErrorTemplate(req.URL.Path); tpl != nil {
			*req = *req.WithContext(ctx)
			return tpl.WithError(errors.RouteNotFound.Label(api.BasePath)) // TODO: api label
		}

		fileHandler, exist := m.hasFileResponse(req)
		if exist {
			*req = *req.WithContext(ctx)
			return fileHandler
		}

		routeMatch, matches = m.match(m.spaRoot, req)

		if !matches {
			if fileHandler != nil {
				return fileHandler
			}

			// Fallback
			*req = *req.WithContext(ctx)
			return m.opts.ServerOptions.ServerErrTpl.WithError(errors.RouteNotFound)
		}
	}

	pathParams := make(request.PathParameter, len(routeMatch.Vars))
	for k, value := range routeMatch.Vars {
		key := strings.TrimSuffix(strings.TrimSuffix(k, "*"), "|.+")
		pathParams[key] = value
	}

	pt, _ := routeMatch.Route.GetPathTemplate()
	p := pt
	for k, v := range routeMatch.Vars {
		p = strings.Replace(p, "{"+k+"}", v, 1)
	}
	wc := strings.TrimPrefix(req.URL.Path, p)
	if strings.HasPrefix(wc, "/") {
		wc = strings.TrimPrefix(wc, "/")
	}

	if wc != "" {
		ctx = context.WithValue(ctx, request.Wildcard, wc)
	}

	ctx = context.WithValue(ctx, request.PathParams, pathParams)
	*req = *req.WithContext(ctx)

	return routeMatch.Handler
}

func (m *Mux) match(root *gmux.Router, req *http.Request) (*gmux.RouteMatch, bool) {
	var routeMatch gmux.RouteMatch
	if root.Match(req, &routeMatch) {
		return &routeMatch, true
	}

	return nil, false
}

func (m *Mux) hasFileResponse(req *http.Request) (http.Handler, bool) {
	routeMatch, matches := m.match(m.fileRoot, req)
	if !matches {
		return nil, false
	}

	fileHandler := routeMatch.Handler
	unprotectedHandler := getChildHandler(fileHandler)
	if fh, ok := unprotectedHandler.(*handler.File); ok {
		return fileHandler, fh.HasResponse(req)
	}

	if fh, ok := fileHandler.(*handler.File); ok {
		return fileHandler, fh.HasResponse(req)
	}

	return fileHandler, false
}

func (m *Mux) getAPIErrorTemplate(reqPath string) (*errors.Template, *config.API) {
	for api, path := range m.opts.ServerOptions.APIBasePaths {
		if !isConfigured(path) {
			continue
		}

		var spaPaths, filesPaths []string

		if len(m.opts.ServerOptions.SPABasePaths) == 0 {
			spaPaths = []string{""}
		} else {
			spaPaths = m.opts.ServerOptions.SPABasePaths
		}

		if len(m.opts.ServerOptions.FilesBasePaths) == 0 {
			filesPaths = []string{""}
		} else {
			filesPaths = m.opts.ServerOptions.FilesBasePaths
		}

		for _, spaPath := range spaPaths {
			for _, filesPath := range filesPaths {
				if isAPIError(path, filesPath, spaPath, reqPath) {
					return m.opts.ServerOptions.APIErrTpls[api], api
				}
			}
		}
	}

	return nil, nil
}

func mustAddRoute(root *gmux.Router, path string, handler http.Handler, trailingSlash bool) {
	if strings.HasSuffix(path, wildcardSearch) {
		path = path[:len(path)-len(wildcardSearch)]
		if len(path) == 0 {
			path = "/" // path at least be /
		}
		root.Path(path).Handler(handler) // register /path ...
		if !strings.HasSuffix(path, "/") {
			path = path + "/" // ... and /path/**
		}
		root.PathPrefix(path).Handler(handler)
		return
	}

	if len(path) == 0 {
		path = "/" // path at least be /
	}
	// cannot use Router.StrictSlash(true) because redirect and subsequent GET request would cause problem with CORS
	if trailingSlash {
		if strings.HasSuffix(path, "/") {
			path = strings.TrimSuffix(path, "/")
		}
		if len(path) > 0 {
			root.Path(path).Handler(handler) // register /path ...
		}
		path = path + "/" // ... and /path/
	}
	root.Path(path).Handler(handler)
}

// isAPIError checks the path w/ and w/o the
// trailing slash against the request path.
func isAPIError(apiPath, filesBasePath, spaBasePath, reqPath string) bool {
	if matchesPath(apiPath, reqPath) {
		if isConfigured(filesBasePath) && apiPath == filesBasePath {
			return false
		}
		if isConfigured(spaBasePath) && apiPath == spaBasePath {
			return false
		}

		return true
	}

	return false
}

// matchesPath checks the path w/ and w/o the
// trailing slash against the request path.
func matchesPath(path, reqPath string) bool {
	p1 := path
	p2 := path

	if p1 != "/" && !strings.HasSuffix(p1, "/") {
		p1 += "/"
	}
	if p2 != "/" && strings.HasSuffix(p2, "/") {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		return true
	}

	return false
}

func isConfigured(basePath string) bool {
	return basePath != ""
}
