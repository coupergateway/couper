package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/config/runtime"
)

var (
	PatternSlashError = errors.New("missing slash in first place")
	WildcardPathError = errors.New("wildcard path must end with /** and has no other occurrences")
)

const (
	wildcardReplacement = "(:?$|/(.*))"
	wildcardSearch      = "/**"
)

type Route struct {
	handler  http.Handler
	matcher  *regexp.Regexp
	pattern  string
	sortLen  int
	wildcard bool
}

func NewRoute(pattern string, handler http.Handler) (*Route, error) {
	if pattern == "" || pattern[0] != '/' {
		return nil, PatternSlashError
	}

	if handler == nil {
		return nil, errors.New("missing handler for route pattern: " + pattern)
	}

	// TODO: parse/create regexp "template" parsing
	matchPattern := "^" + pattern
	if !validWildcardPath(matchPattern) {
		return nil, WildcardPathError
	}

	sortLen := len(strings.ReplaceAll(pattern, wildcardSearch, "/"))
	if !strings.HasSuffix(pattern, wildcardSearch) && !strings.HasSuffix(pattern, "$") {
		matchPattern = matchPattern + "$"
	}

	matchPattern = strings.ReplaceAll(matchPattern, wildcardSearch, wildcardReplacement)
	matcher := regexp.MustCompile(matchPattern)
	return &Route{
		handler:  handler,
		matcher:  matcher,
		pattern:  pattern,
		sortLen:  sortLen,
		wildcard: strings.HasSuffix(matchPattern, wildcardReplacement),
	}, nil
}

func (r *Route) Match(req *http.Request) http.Handler {
	if r.matcher.MatchString(req.URL.Path) {
		if r.wildcard {
			match := r.matcher.FindStringSubmatch(req.URL.Path)
			if len(match) > 1 {
				*req = *req.WithContext(context.WithValue(req.Context(), config.WildcardCtxKey, match[1]))
			}
		}
		return r.handler
	}
	return nil
}

func (r *Route) Pattern() string {
	return r.matcher.String()
}

type routes []*Route

func (r routes) add(pattern string, h http.Handler) routes {
	route, err := NewRoute(pattern, h)
	if err != nil {
		panic(err)
	}

	n := len(r)
	idx := sort.Search(n, func(i int) bool {
		return (r[i].sortLen) < (route.sortLen)
	})
	if idx == n {
		return append(r, route)
	}

	routes := append(r, &Route{})      // try to grow the slice in place, any entry works.
	copy(routes[idx+1:], routes[idx:]) // Move shorter entries down
	routes[idx] = route
	return routes
}

type routesMap map[string]routes

func (m routesMap) Add(domain, pattern string, h http.Handler) routesMap {
	_, ok := m[domain]
	if !ok {
		m[domain] = make(routes, 0)
	}
	m[domain] = m[domain].add(pattern, h)
	return m
}

// Match searches for explicit domain paths first and finally the wildcard ones.
func (m routesMap) Match(domain string, req *http.Request) (http.Handler, bool) {
	var wildcardRoutes routes

	r, ok := m[domain]
	if !ok {
		return nil, false
	}

	for _, route := range r {
		if route.wildcard {
			wildcardRoutes = append(wildcardRoutes, route)
			continue
		}
		if h := route.Match(req); h != nil {
			*req = *req.WithContext(context.WithValue(req.Context(), runtime.Endpoint, route.pattern))
			return h, true
		}
	}

	for _, route := range wildcardRoutes {
		if h := route.Match(req); h != nil {
			return h, true
		}
	}

	return nil, false
}

func validWildcardPath(path string) bool {
	if cnt := strings.Count(path, wildcardSearch); cnt > 1 ||
		(cnt == 1) && !strings.HasSuffix(path, wildcardSearch) {
		return false
	}
	return true
}
