package server

import (
	"context"
	"net/http"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/runtime"
)

type Router struct {
	routes config.Routes
}

func NewRouter(routes config.Routes) *Router {
	return &Router{routes: routes}
}

// Match searches for explicit pathes first and finally the wildcard ones.
func (r *Router) Match(req *http.Request) (http.Handler, bool) {
	var wildcardRoutes config.Routes

	if len(r.routes) == 0 {
		return nil, false
	}

	for _, route := range r.routes {
		if route.HasWildcard() {
			wildcardRoutes = append(wildcardRoutes, route)
			continue
		}
		if h := r.match(req, route); h != nil {
			*req = *req.WithContext(context.WithValue(req.Context(), runtime.Endpoint, route.Name()))
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

func (r *Router) match(req *http.Request, route *config.Route) http.Handler {
	if route.GetMatcher().MatchString(req.URL.Path) {
		if route.HasWildcard() {
			match := route.GetMatcher().FindStringSubmatch(req.URL.Path)
			if len(match) > 1 {
				*req = *req.WithContext(context.WithValue(req.Context(), config.WildcardCtxKey, match[1]))
			}
		}
		return route.GetHandler()
	}
	return nil
}
