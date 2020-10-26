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
	apiPath        string
	apiErrHandler  *errors.Template
	endpointRoot   *pathpattern.Node
	fileBasePath   string
	fileHandler    http.Handler
	fileErrHandler *errors.Template
	router         *openapi3filter.Router
	spaRoot        *pathpattern.Node
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

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = &runtime.MuxOptions{
			APIErrTpl:  errors.DefaultJSON,
			FileErrTpl: errors.DefaultHTML,
		}
	}

	mux := &Mux{
		apiPath:        opts.APIPath,
		apiErrHandler:  opts.APIErrTpl,
		fileBasePath:   opts.FileBasePath,
		fileHandler:    opts.FileHandler,
		fileErrHandler: opts.FileErrTpl,
		endpointRoot:   &pathpattern.Node{},
		spaRoot:        &pathpattern.Node{},
	}

	for path, h := range opts.EndpointRoutes {
		// TODO: handle method option per endpoint configuration
		mux.mustAddRoute(mux.endpointRoot, allowedMethods, path, h)
	}

	for path, h := range opts.SPARoutes {
		mux.mustAddRoute(mux.spaRoot, []string{http.MethodGet}, path, h)
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

	for _, method := range methods {
		pathOptions := &pathpattern.Options{}

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
		if strings.HasPrefix(req.URL.Path, m.apiPath) {
			return m.apiErrHandler.ServeError(errors.APIRouteNotFound)
		}

		if m.hasFileResponse(req) {
			return m.fileHandler
		}

		node, paramValues = m.match(m.spaRoot, req)
		if node == nil {
			if m.fileHandler != nil && strings.HasPrefix(req.URL.Path, m.fileBasePath) {
				return m.fileErrHandler.ServeError(errors.FilesRouteNotFound)
			}
			// TODO: server err handler
			if m.fileErrHandler != nil {
				return m.fileErrHandler.ServeError(errors.Configuration)
			}
			return errors.DefaultHTML.ServeError(errors.Configuration)
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
	// no hosts are configured, lookup /wo hostPath first
	node, paramValues := root.Match(matchPath)
	if node == nil {
		hostPath := pathpattern.PathFromHost(req.Host, false)
		matchPath = req.Method + " " + utils.JoinPath(hostPath, req.URL.Path)
		node, paramValues = root.Match(matchPath)
	}
	return node, paramValues
}

func (m *Mux) hasFileResponse(req *http.Request) bool {
	if m.fileHandler == nil {
		return false
	}

	fileHandler := m.fileHandler
	if p, isProtected := m.fileHandler.(ac.ProtectedHandler); isProtected {
		fileHandler = p.Child()
	}
	if fh, ok := fileHandler.(handler.HasResponse); ok && fh.HasResponse(req) {
		return true
	}

	return false
}
