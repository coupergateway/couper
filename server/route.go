package server

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
)

var PatternSlashError = errors.New("missing slash in first place")

type Route struct {
	handler http.Handler
	matcher *regexp.Regexp
	parent  *Route
	pattern string
	sub     *Route
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
	matchPattern = strings.ReplaceAll(matchPattern, "/**", "/.*")
	matcher := regexp.MustCompile(matchPattern)
	return &Route{
		handler: handler,
		matcher: matcher,
		pattern: pattern,
	}, nil

}

func (r *Route) Match(req *http.Request) http.Handler {
	if r.matcher.MatchString(req.URL.Path) {
		return r.handler
	}
	return nil
}

func (r *Route) Pattern() string {
	return r.matcher.String()
}

func (r *Route) Sub(pattern string, handler http.Handler) (*Route, error) {
	route, err := NewRoute(pattern, handler)
	if err != nil {
		return nil, err
	}
	route.parent = r
	r.sub = route
	return route, nil
}
