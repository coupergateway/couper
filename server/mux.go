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

const serverOptionsKey = "serverContextOptions"

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = runtime.NewMuxOptions(nil)
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

	node, srvCtxOpts, paramValues := m.match(m.endpointRoot, req)
	if node == nil {
		// No matches for api or free endpoints. Determine if we have entered an api basePath
		// and handle api related errors accordingly.
		// Otherwise look for existing files or spa fallback.
		if tpl := getAPIErrorTemplate(srvCtxOpts, req.URL.Path); tpl != nil {
			return tpl.ServeError(errors.APIRouteNotFound)
		}

		fileHandler, fileSrvCtxOpts, exist := m.hasFileResponse(req)
		if exist {
			return fileHandler
		}
		if fileSrvCtxOpts != nil && srvCtxOpts == nil {
			srvCtxOpts = fileSrvCtxOpts
		}

		var spaSrvCtxOpts *server.Options
		node, spaSrvCtxOpts, paramValues = m.match(m.spaRoot, req)
		if spaSrvCtxOpts != nil && srvCtxOpts == nil {
			srvCtxOpts = spaSrvCtxOpts
		}

		if node == nil {
			// no spa path?
			if fileSrvCtxOpts != nil && isConfigured(fileSrvCtxOpts.FileBasePath) && isFileError(srvCtxOpts, req.URL.Path) {
				return fileSrvCtxOpts.FileErrTpl.ServeError(errors.FilesRouteNotFound)
			}

			if srvCtxOpts != nil {
				return srvCtxOpts.ServerErrTpl.ServeError(errors.Configuration)
			}
			// Fallback
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

func (m *Mux) match(root *pathpattern.Node, req *http.Request) (*pathpattern.Node, *server.Options, []string) {
	hostPath := pathpattern.PathFromHost(req.Host, false)
	matchHostPath := req.Method + " " + utils.JoinPath(hostPath, req.URL.Path)
	node, paramValues := root.Match(matchHostPath)
	if _, ok := m.opts.Hosts[req.Host]; !ok && node == nil { // no specific hosts found, lookup for general path matches
		matchPath := req.Method + " " + req.URL.Path
		node, paramValues = root.Match(matchPath)
	}

	var srvCtxOpts *server.Options

	if node == nil { // still no match, try to obtain some server configuration from suffixList
		hostPathIdx := strings.IndexByte(matchHostPath, '/')
	pathLoop:
		for _, matchPath := range []string{matchHostPath[:hostPathIdx], req.Method + " "} {
			for _, suffix := range root.Suffixes {
				if suffix.Pattern == matchPath {
					if len(suffix.Node.Suffixes) > 0 {
						// FIXME some improvement, filter health routes, nested search etc. maybe mark on route.add
						for i := len(suffix.Node.Suffixes[0].Node.Suffixes); i > 0; i-- {
							srvCtxOpts = unwrapServerOptions(suffix.Node.Suffixes[0].Node.Suffixes[i-1])
							if srvCtxOpts != nil {
								break
							}
						}
					} else {
						srvCtxOpts = unwrapServerOptions(pathpattern.Suffix{Node: suffix.Node})
					}
					break pathLoop
				}
			}
		}
	}

	if node != nil && node.Value != nil {
		srvCtxOpts = unwrapServerOptions(pathpattern.Suffix{Node: node})
	}

	if srvCtxOpts != nil {
		*req = *req.WithContext(context.WithValue(req.Context(), request.ServerName, srvCtxOpts.ServerName))
	}

	return node, srvCtxOpts, paramValues
}

func (m *Mux) hasFileResponse(req *http.Request) (http.Handler, *server.Options, bool) {
	node, srvCtxOpts, _ := m.match(m.fileRoot, req)
	if node == nil {
		return nil, srvCtxOpts, false
	}

	route := node.Value.(*openapi3filter.Route)
	fileHandler := route.Handler
	if p, isProtected := fileHandler.(ac.ProtectedHandler); isProtected {
		fileHandler = p.Child()
		srvCtxOpts = p.Child().(server.Context).Options()
	}

	if fh, ok := fileHandler.(handler.HasResponse); ok {
		return fileHandler, srvCtxOpts, fh.HasResponse(req)
	}

	return fileHandler, srvCtxOpts, false
}

func unwrapServerOptions(suffix pathpattern.Suffix) *server.Options {
	if suffix.Node.Value != nil {
		if patternNode, ok := suffix.Node.Value.(*openapi3filter.Route); ok {
			if patternNode.Server != nil {
				return patternNode.Server.Variables[serverOptionsKey].Default.(*server.Options)
			}
		}
	}
	return unwrapServerOptions(suffix.Node.Suffixes[len(suffix.Node.Suffixes)-1]) // FIXME: check other more explicit suffixes?
}

func getAPIErrorTemplate(srvOptions *server.Options, reqPath string) *errors.Template {
	if srvOptions == nil {
		return nil
	}

	for i, path := range srvOptions.APIBasePath {
		if !isConfigured(path) {
			continue
		}

		if isAPIError(srvOptions, path, reqPath) {
			return srvOptions.APIErrTpl[i]
		}
	}

	return nil
}

// isAPIError checks the path w/ and w/o the
// trailing slash against the request path.
func isAPIError(srvOpts *server.Options, apiPath, reqPath string) bool {
	if srvOpts == nil {
		return false
	}
	p1 := apiPath
	p2 := apiPath

	if p1 != "/" && !strings.HasSuffix(p1, "/") {
		p1 += "/"
	}
	if p2 != "/" && strings.HasSuffix(p2, "/") {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		if isConfigured(srvOpts.FileBasePath) && apiPath == srvOpts.FileBasePath {
			return false
		}
		if isConfigured(srvOpts.SPABasePath) && apiPath == srvOpts.SPABasePath {
			return false
		}

		return true
	}

	return false
}

// isFileError checks the path w/ and w/o the
// trailing slash against the request path.
func isFileError(srvOpts *server.Options, reqPath string) bool {
	if srvOpts == nil {
		return false
	}
	p1 := srvOpts.FileBasePath
	p2 := srvOpts.FileBasePath

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
