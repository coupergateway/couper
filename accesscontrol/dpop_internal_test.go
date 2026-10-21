package accesscontrol

import (
	"net/url"
	"strings"
	"testing"
)

// legacyNormalize is the implementation before the router normalizer was shared.
// It is the reference for the differential test below.
func legacyNormalize(u *url.URL) (*url.URL, error) {
	ru, err := url.Parse(u.RequestURI())
	if err != nil {
		return nil, err
	}
	u = u.ResolveReference(ru)
	u.RawPath = u.Path

	hostname := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" ||
		u.Scheme == "http" && port == "80" ||
		u.Scheme == "https" && port == "443" {
		u.Host = hostname
	} else {
		u.Host = hostname + ":" + port
	}
	return u, nil
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		// scheme-based normalization
		{"http://host", "http://host/", false},
		{"http://host:80", "http://host/", false},
		{"https://host:443/", "https://host/", false},
		{"https://host:/", "https://host/", false},
		{"https://Host.Example:8443/Path", "https://host.example:8443/Path", false},
		{"http://user@host/", "http://user@host/", false},
		// path segment normalization, same rules as the router
		{"http://host/a/", "http://host/a/", false},
		{"http://host/a//b", "http://host/a/b", false},
		{"http://host//a", "http://host/a", false},
		{"http://host/a/./b", "http://host/a/b", false},
		{"http://host/a/../b", "http://host/b", false},
		{"http://host/a/%2e%2e/b", "http://host/b", false},
		{"http://host/a//../b", "http://host/b", false},
		{"http://host/a/..", "http://host/", false},
		{"http://host/a/b/..", "http://host/a/", false},
		{"http://host/..", "", true},
		// percent-encoding normalization
		{"http://host/%63/%7bx%7d", "http://host/c/%7Bx%7D", false},
		{"http://host/a%2Fb", "http://host/a/b", false},
		{"http://host/a%20b", "http://host/a%20b", false},
		// query and fragment stay as they are, the proof must not carry them
		{"http://host/path?q=1", "http://host/path?q=1", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(subT *testing.T) {
			u, err := url.Parse(tt.in)
			if err != nil {
				subT.Fatal(err)
			}
			got, err := normalize(u)
			if (err != nil) != tt.wantErr {
				subT.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			if got.String() != tt.want {
				subT.Errorf("want %q, got %q", tt.want, got.String())
			}
		})
	}
}

// FuzzNormalizeMatchesLegacy guards against lost behavior: outside the segments
// the router rules change on purpose, the result must equal the previous one.
func FuzzNormalizeMatchesLegacy(f *testing.F) {
	for _, seed := range []string{
		"http://host", "http://host/", "http://host:80", "https://HOST:443/a/b/", "https://host:/",
		"http://host/%63/%7bx%7d", "http://host/a%2Fb", "http://host/a%20b", "http://user@host:8080/a",
		"http://host/a/b/c", "http://host/a.b/c..d/", "http://host/%2", "http://[::1]:8080/a",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		u, err := url.Parse(raw)
		if err != nil || u.Opaque != "" || u.Host == "" {
			return
		}
		// Query and fragment are not part of the comparison; the parse path must
		// not fail on them either.
		u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = "", false, "", ""
		if hasRouterNormalizedSegment(u.Path) {
			return
		}

		legacy, lerr := legacyNormalize(cloneURL(u))
		got, gerr := normalize(cloneURL(u))
		if (lerr != nil) != (gerr != nil) {
			t.Fatalf("%q: legacy err %v, got err %v", raw, lerr, gerr)
		}
		if lerr != nil {
			return
		}
		if legacy.String() != got.String() {
			t.Fatalf("%q: legacy %q, got %q", raw, legacy.String(), got.String())
		}
	})
}

// hasRouterNormalizedSegment reports whether the decoded path contains an empty
// or dot segment, the cases where the shared normalizer differs on purpose.
func hasRouterNormalizedSegment(path string) bool {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, segment := range segments {
		switch segment {
		case ".", "..":
			return true
		case "":
			if i < len(segments)-1 {
				return true
			}
		}
	}
	return false
}

func cloneURL(u *url.URL) *url.URL {
	c := *u
	return &c
}
