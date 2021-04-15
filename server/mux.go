package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/pathpattern"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/config/runtime/server"
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

const serverOptionsKey = "serverContextOptions"

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = runtime.NewMuxOptions()
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

		var serverOpts *server.Options
		if optsHandler, ok := handler.(server.Context); ok {
			serverOpts = optsHandler.Options()
		}

		node.Value = &openapi3filter.Route{
			Method:  method,
			Path:    path,
			Handler: handler,
			Server: &openapi3.Server{Variables: map[string]*openapi3.ServerVariable{
				serverOptionsKey: {Default: serverOpts},
			}},
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
		if tpl, api := m.getAPIErrorTemplate(req.URL.Path); tpl != nil {
			return tpl.ServeError(errors.RouteNotFound.Label(api.BasePath)) // TODO: api label
		}

		fileHandler, exist := m.hasFileResponse(req)
		if exist {
			return fileHandler
		}

		node, paramValues = m.match(m.spaRoot, req)

		if node == nil {
			if fileHandler != nil {
				return fileHandler
			}

			if isConfigured(m.opts.ServerOptions.FilesBasePath) && matchesPath(m.opts.ServerOptions.FilesBasePath, req.URL.Path) {
				return m.opts.ServerOptions.FilesErrTpl.ServeError(errors.RouteNotFound)
			}

			// Fallback
			return m.opts.ServerOptions.ServerErrTpl.ServeError(errors.Configuration)
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
	*req = *req.WithContext(context.WithValue(req.Context(), request.ServerName, m.opts.ServerOptions.ServerName))

	return root.Match(req.Method + " " + req.URL.Path)
}

func (m *Mux) hasFileResponse(req *http.Request) (http.Handler, bool) {
	node, _ := m.match(m.fileRoot, req)
	if node == nil {
		return nil, false
	}

	route := node.Value.(*openapi3filter.Route)
	fileHandler := route.Handler
	if p, isProtected := fileHandler.(ac.ProtectedHandler); isProtected {
		if fh, ok := p.Child().(handler.HasResponse); ok {
			return fileHandler, fh.HasResponse(req)
		}
	}

	if fh, ok := fileHandler.(handler.HasResponse); ok {
		return fileHandler, fh.HasResponse(req)
	}

	return fileHandler, false
}

func (m *Mux) getAPIErrorTemplate(reqPath string) (*errors.Template, *config.API) {
	for api, path := range m.opts.ServerOptions.APIBasePaths {
		if !isConfigured(path) {
			continue
		}

		if isAPIError(path, m.opts.ServerOptions.FilesBasePath, m.opts.ServerOptions.SPABasePath, reqPath) {
			return m.opts.ServerOptions.APIErrTpls[api], api
		}
	}

	return nil, nil
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
