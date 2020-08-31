package server

import (
	"context"
	"net/http"

	"go.avenga.cloud/couper/gateway/config/request"
	"go.avenga.cloud/couper/gateway/config/runtime"
)

// Router represents the Router object.
type Router struct {
	routes runtime.Routes
}

// NewRouter creates a new Router object.
func NewRouter(routes runtime.Routes) *Router {
	return &Router{routes: routes}
}

// Match searches for explicit pathes first and finally the wildcard ones.
func (r *Router) Match(req *http.Request) (http.Handler, bool) {
	var wildcardRoutes runtime.Routes

	if len(r.routes) == 0 {
		return nil, false
	}

	for _, route := range r.routes {
		if route.HasWildcard() {
			wildcardRoutes = append(wildcardRoutes, route)
			continue
		}
		if h := r.match(req, route); h != nil {
			*req = *req.WithContext(context.WithValue(req.Context(), request.Endpoint, route.Name()))
			return h, true
		}
	}

	for _, route := range wildcardRoutes {
		if h := r.match(req, route); h != nil {
			return h, true
		}
	}

	return nil, false
}

func (r *Router) match(req *http.Request, route *runtime.Route) http.Handler {
	if route.GetMatcher().MatchString(req.URL.Path) {
		if route.HasWildcard() {
			match := route.GetMatcher().FindStringSubmatch(req.URL.Path)
			if len(match) > 1 {
				*req = *req.WithContext(context.WithValue(req.Context(), request.Wildcard, match[1]))
			}
		}
		return route.GetHandler()
	}
	return nil
}
