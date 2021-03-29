package runtime

import (
	"testing"

	"github.com/avenga/couper/config"
)

func TestServer_isUnique(t *testing.T) {
	endpoints := make(map[string]bool)

	type testCase struct {
		pattern string
		clean   string
		unique  bool
	}

	for i, tc := range []testCase{
		{"/abc", "/abc", true},
		{"/abc", "/abc", false},
		{"/abc/", "/abc/", true},
		{"/x/{xxx}", "/x/{}", true},
		{"/x/{yyy}", "/x/{}", false},
		{"/x/{xxx}/a/{yyy}", "/x/{}/a/{}", true},
		{"/x/{yyy}/a/{xxx}", "/x/{}/a/{}", false},
	} {
		unique, cleanPattern := isUnique(endpoints, tc.pattern)
		endpoints[cleanPattern] = true

		if unique != tc.unique {
			t.Errorf("%d: Unexpected unique status given: %t", i+1, unique)
		}
		if cleanPattern != tc.clean {
			t.Errorf("%d: Unexpected cleanPattern given: %s, want %s", i+1, cleanPattern, tc.clean)
		}
	}
}

func TestServer_getEndpointsList(t *testing.T) {
	srvConf := &config.Server{
		APIs: []*config.API{
			{
				Endpoints: []*config.Endpoint{
					{Pattern: "/api/1"},
					{Pattern: "/api/2"},
				},
			},
		},
		Endpoints: []*config.Endpoint{
			{Pattern: "/free/1"},
			{Pattern: "/free/2"},
		},
	}

	endpoints := newEndpointMap(srvConf)
	if l := len(endpoints); l != 4 {
		t.Fatalf("Expected 4 endpointes, given %d", l)
	}

	checks := map[string]HandlerKind{
		"/api/1":  api,
		"/api/2":  api,
		"/free/1": endpoint,
		"/free/2": endpoint,
	}

	for pattern, kind := range checks {
		var exist bool
		for endpoint, parent := range endpoints {
			if endpoint.Pattern == pattern {
				if kind == api && parent == nil {
					t.Errorf("Expected an api endpoint for path pattern: %q", pattern)
				}
				exist = true
				break
			}
		}
		if !exist {
			t.Errorf("Expected an endpoint for path pattern: %q", pattern)
		}
	}
}
