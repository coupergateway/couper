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
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
)

// Mux is a http request router and dispatches requests
// to their corresponding http handlers.
type Mux struct {
	apiPath     string
	fileHandler http.Handler
	root        *pathpattern.Node
	router      *openapi3filter.Router
	spaHandler  http.Handler
}

func NewMux(apiPath string, fileHandler, spaHandler http.Handler) *Mux {
	return &Mux{
		apiPath:     apiPath,
		fileHandler: fileHandler,
		spaHandler:  spaHandler,
		root:        &pathpattern.Node{},
	}
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

func (m *Mux) MustAddRoute(path string, handler http.Handler) *Mux {
	const wildcardReplacement = "/{_couper_wildcardMatch*}"
	const wildcardSearch = "/**"

	// TODO: handle method option per endpoint
	// TODO: ensure uppercase method string if passed as argument
	for _, method := range allowedMethods {
		pathOptions := &pathpattern.Options{}

		if strings.HasSuffix(path, "/**") {
			pathOptions.SupportRegExp = true
			path = path[:len(path)-len(wildcardSearch)] + wildcardReplacement
		}

		node, err := m.root.CreateNode(method+" "+path, pathOptions)
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
	node, paramValues := m.root.Match(req.Method + " " + req.URL.Path)
	if node != nil {
		route, _ = node.Value.(*openapi3filter.Route)
	} else {
		// No matches for api or free endpoints. Determine if we have entered an api basePath
		// and handle api related errors accordingly.
		// Otherwise look for existing files or spa fallback.
		apiPath := req.URL.Path
		for strings.HasSuffix(apiPath, "/") {
			apiPath = apiPath[:len(apiPath)-1]
		}

		if apiPath == m.apiPath {
			// TODO configured tpl
			return errors.DefaultJSON.ServeError(errors.APIRouteNotFound)
		}

		if m.fileHandler != nil {
			fileHandler := m.fileHandler
			if p, isProtected := m.fileHandler.(ac.ProtectedHandler); isProtected {
				fileHandler = p.Child()
			}
			if fh, ok := fileHandler.(handler.HasResponse); ok && fh.HasResponse(req) {
				return m.fileHandler
			}
		}

		if m.spaHandler != nil { // TODO: match paths []
			return m.spaHandler
		}
		return nil // TODO: is fileError?
	}

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
