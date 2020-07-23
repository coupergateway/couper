package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

var (
	PatternSlashError = errors.New("missing slash in first place")
	WildcardPathError = errors.New("wildcard path must end with /** and has no other occurrences")
)

const wildcardCtx = "route_wildcard"

type Route struct {
	handler  http.Handler
	matcher  *regexp.Regexp
	pattern  string
	sortLen  int
	wildcard bool
}

func NewRoute(pattern string, handler http.Handler) (*Route, error) {
	const wildcardReplacement = "/(.*)"
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
	matchPattern = strings.ReplaceAll(matchPattern, "/**", wildcardReplacement)
	matcher := regexp.MustCompile(matchPattern)
	return &Route{
		handler:  handler,
		matcher:  matcher,
		pattern:  pattern,
		sortLen:  len(strings.ReplaceAll(matchPattern, "/**", "/")),
		wildcard: strings.HasSuffix(matchPattern, wildcardReplacement),
	}, nil

}

func (r *Route) Match(req *http.Request) http.Handler {
	if r.matcher.MatchString(req.URL.Path) {
		if r.wildcard {
			match := r.matcher.FindStringSubmatch(req.URL.Path)
			if len(match) > 1 {
				*req = *req.WithContext(context.WithValue(req.Context(), wildcardCtx, match[1]))
			}
		}
		return r.handler
	}
	return nil
}

func (r *Route) Pattern() string {
	return r.matcher.String()
}

func validWildcardPath(path string) bool {
	if cnt := strings.Count(path, "/**"); cnt > 1 ||
		(cnt == 1) && !strings.HasSuffix(path, "/**") {
		return false
	}
	return true
}
