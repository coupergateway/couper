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

const (
	wildcardCtx         = "route_wildcard"
	wildcardReplacement = "/(.*)"
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
	if cnt := strings.Count(path, wildcardSearch); cnt > 1 ||
		(cnt == 1) && !strings.HasSuffix(path, wildcardSearch) {
		return false
	}
	return true
}
