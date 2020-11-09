package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/pathpattern"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/utils"
)

// Mux is a http request router and dispatches requests
// to their corresponding http handlers.
type Mux struct {
	endpointRoot *pathpattern.Node
	fileRoot     *pathpattern.Node
	opts         *runtime.MuxOptions
	router       *openapi3filter.Router
	spaRoot      *pathpattern.Node
}

var allowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

var fileMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = &runtime.MuxOptions{
			APIErrTpl:    errors.DefaultJSON,
			FileErrTpl:   errors.DefaultHTML,
			ServerErrTpl: errors.DefaultHTML,
		}
	}

	mux := &Mux{
		opts:         opts,
		endpointRoot: &pathpattern.Node{},
		fileRoot:     &pathpattern.Node{},
		spaRoot:      &pathpattern.Node{},
	}

	for path, h := range opts.EndpointRoutes {
		// TODO: handle method option per endpoint configuration
		mux.mustAddRoute(mux.endpointRoot, allowedMethods, path, h)
	}

	for path, h := range opts.FileRoutes {
		mux.mustAddRoute(mux.fileRoot, fileMethods, utils.JoinPath(path, "/**"), h)
	}

	for path, h := range opts.SPARoutes {
		mux.mustAddRoute(mux.spaRoot, fileMethods, path, h)
	}

	return mux
}

func (m *Mux) MustAddRoute(method, path string, handler http.Handler) *Mux {
	methods := allowedMethods[:]
	if method != "*" {
		um := strings.ToUpper(method)
		var allowed bool
		for _, am := range allowedMethods {
			if um == am {
				allowed = true
				break
			}
		}
		if !allowed {
			panic(fmt.Errorf("method not allowed: %q, path: %q", um, path))
		}

		methods = []string{um}
	}
	return m.mustAddRoute(m.endpointRoot, methods, path, handler)
}

func (m *Mux) mustAddRoute(root *pathpattern.Node, methods []string, path string, handler http.Handler) *Mux {
	const wildcardReplacement = "/{_couper_wildcardMatch*}"
	const wildcardSearch = "/**"

	// TODO: Unique Options per method if configurable later on
	pathOptions := &pathpattern.Options{}

	for _, method := range methods {
		if strings.HasSuffix(path, wildcardSearch) {
			pathOptions.SupportWildcard = true
			path = path[:len(path)-len(wildcardSearch)] + wildcardReplacement
		}

		node, err := root.CreateNode(method+" "+path, pathOptions)
		if err != nil {
			panic(fmt.Errorf("create path node failed: %s %q: %v", method, path, err))
		}

		node.Value = &openapi3filter.Route{
			Method:  method,
			Path:    path,
			Handler: handler,
		}
	}
	return m
}

func (m *Mux) FindHandler(req *http.Request) http.Handler {
	var route *openapi3filter.Route

	node, paramValues := m.match(m.endpointRoot, req)
	if node == nil {
		// No matches for api or free endpoints. Determine if we have entered an api basePath
		// and handle api related errors accordingly.
		// Otherwise look for existing files or spa fallback.
		if isConfigured(m.opts.APIBasePath) && m.isAPIError(req.URL.Path) {
			return m.opts.APIErrTpl.ServeError(errors.APIRouteNotFound)
		}

		fileHandler, exist := m.hasFileResponse(req)
		if exist {
			return fileHandler
		}

		node, paramValues = m.match(m.spaRoot, req)
		if node == nil {
			if isConfigured(m.opts.FileBasePath) && m.isFileError(req.URL.Path) {
				return m.opts.FileErrTpl.ServeError(errors.FilesRouteNotFound)
			}

			return m.opts.ServerErrTpl.ServeError(errors.Configuration)
		}
	}

	route, _ = node.Value.(*openapi3filter.Route)

	pathParams := make(request.PathParameter, len(paramValues))
	paramKeys := node.VariableNames
	for i, value := range paramValues {
		key := paramKeys[i]
		if strings.HasSuffix(key, "*") {
			key = key[:len(key)-1]
		}
		pathParams[key] = value
	}

	ctx := req.Context()

	const wcm = "_couper_wildcardMatch"
	if wildcardMatch, ok := pathParams[wcm]; ok {
		ctx = context.WithValue(ctx, request.Wildcard, wildcardMatch)
		delete(pathParams, wcm)
	}

	ctx = context.WithValue(ctx, request.PathParams, pathParams)
	*req = *req.Clone(ctx)

	return route.Handler
}

func (m *Mux) match(root *pathpattern.Node, req *http.Request) (*pathpattern.Node, []string) {
	matchPath := req.Method + " " + req.URL.Path
	// no hosts are configured, lookup w/o hostPath first
	node, paramValues := root.Match(matchPath)
	if node == nil {
		hostPath := pathpattern.PathFromHost(req.Host, false)
		matchPath = req.Method + " " + utils.JoinPath(hostPath, req.URL.Path)
		node, paramValues = root.Match(matchPath)
	}
	return node, paramValues
}

func (m *Mux) hasFileResponse(req *http.Request) (http.Handler, bool) {
	node, _ := m.match(m.fileRoot, req)
	if node == nil {
		return nil, false
	}

	route := node.Value.(*openapi3filter.Route)
	fileHandler := route.Handler
	if p, isProtected := fileHandler.(ac.ProtectedHandler); isProtected {
		fileHandler = p.Child()
	}

	if fh, ok := fileHandler.(handler.HasResponse); ok {
		return fileHandler, fh.HasResponse(req)
	}

	return fileHandler, false
}

// isAPIError checks the path w/ and w/o the
// trailing slash against the request path.
func (m *Mux) isAPIError(reqPath string) bool {
	p1 := m.opts.APIBasePath
	p2 := m.opts.APIBasePath

	if p1 != "/" && !strings.HasSuffix(p1, "/") {
		p1 += "/"
	}
	if p2 != "/" && strings.HasSuffix(p2, "/") {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		if isConfigured(m.opts.FileBasePath) && m.opts.APIBasePath == m.opts.FileBasePath {
			return false
		}
		if isConfigured(m.opts.SPABasePath) && m.opts.APIBasePath == m.opts.SPABasePath {
			return false
		}

		return true
	}

	return false
}

// isFileError checks the path w/ and w/o the
// trailing slash against the request path.
func (m *Mux) isFileError(reqPath string) bool {
	p1 := m.opts.FileBasePath
	p2 := m.opts.FileBasePath

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
