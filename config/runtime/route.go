package runtime

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

var (
	errPatternSlash = errors.New("missing slash in first place")
	errWildcardPath = errors.New("wildcard path must end with /** and has no other occurrences")
)

const (
	wildcardReplacement = "(:?$|/(.*))"
	wildcardSearch      = "/**"
)

// Route represents the Route object.
type Route struct {
	handler  http.Handler
	matcher  *regexp.Regexp
	pattern  string
	sortLen  int
	wildcard bool
}

// Routes represents the list of Route objects.
type Routes []*Route

// NewRoute creates a new Route object.
func NewRoute(pattern string, handler http.Handler) (*Route, error) {
	if pattern == "" || pattern[0] != '/' {
		return nil, errPatternSlash
	}

	if handler == nil {
		return nil, errors.New("missing handler for route pattern: " + pattern)
	}

	// TODO: parse/create regexp "template" parsing
	matchPattern := "^" + pattern
	if !validWildcardPath(matchPattern) {
		return nil, errWildcardPath
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

// Add adds a new Route to the Routes list.
func (r Routes) Add(pattern string, h http.Handler) Routes {
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

	Routes := append(r, &Route{})      // try to grow the slice in place, any entry works.
	copy(Routes[idx+1:], Routes[idx:]) // Move shorter entries down
	Routes[idx] = route
	return Routes
}

// HasWildcard returns the state of the wildcard flag.
func (r *Route) HasWildcard() bool {
	return r.wildcard
}

// GetHandler returns the HTTP handler.
func (r *Route) GetHandler() http.Handler {
	return r.handler
}

// GetMatcher returns the regexp matcher.
func (r *Route) GetMatcher() *regexp.Regexp {
	return r.matcher
}

func (r *Route) Name() string {
	return r.pattern
}

func validWildcardPath(path string) bool {
	if cnt := strings.Count(path, wildcardSearch); cnt > 1 ||
		(cnt == 1) && !strings.HasSuffix(path, wildcardSearch) {
		return false
	}
	return true
}
