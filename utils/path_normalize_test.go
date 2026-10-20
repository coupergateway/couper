package utils

import (
	"net/url"
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"/", "/", false},
		{"", "", false},
		{"*", "*", false},
		{"/a/b/c", "/a/b/c", false},
		{"/a//b", "/a/b", false},
		{"//a", "/a", false},
		{"//", "/", false},
		{"/a/./b", "/a/b", false},
		{"/./a", "/a", false},
		{"/a/b/../c", "/a/c", false},
		{"/a/b/c/./../../g", "/a/g", false},
		{"/a/b/", "/a/b/", false},
		{"/a/b//", "/a/b/", false},
		{"/a/b/.", "/a/b/", false},
		{"/a/b/..", "/a/", false},
		{"/a/..", "/", false},
		{"/.", "/", false},
		{"/..", "", true},
		{"/a/../..", "", true},
		{"/%2e%2e/a", "", true},
		{"/a/%2E%2E/b", "/b", false},
		{"/a/%2e%2E/b", "/b", false},
		{"/.%2e/a", "", true},
		{"/a/.%2E/b", "/b", false},
		{"/a/%2e./b", "/b", false},
		{"/a/%2e/b", "/a/b", false},
		{"/a/%2e%2e%2e/b", "/a/%2e%2e%2e/b", false},
		{"/a/.../b", "/a/.../b", false},
		{"/a%2Eb", "/a%2Eb", false},
		{"/%61dmin", "/%61dmin", false},
		{"/a%2fb", "/a%2fb", false},
		{"/a%2F..%2Fb", "/a%2F..%2Fb", false},
		{"/a%2F%2Fb", "/a%2F%2Fb", false},
		{"/a%20b", "/a%20b", false},
		{`/a\..\b`, `/a\..\b`, false},
		{"/a%5C..%5Cb", "/a%5C..%5Cb", false},
		{"/a%25", "/a%25", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(subT *testing.T) {
			got, err := NormalizePath(tt.in)
			if (err != nil) != tt.wantErr {
				subT.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if got != tt.want {
				subT.Errorf("want %q, got %q", tt.want, got)
			}
			if err != nil {
				return
			}
			again, err := NormalizePath(got)
			if err != nil || again != got {
				subT.Errorf("not idempotent: %q -> %q, %v", got, again, err)
			}
		})
	}
}

func TestNormalizeURLPath(t *testing.T) {
	tests := []struct {
		in          string
		wantPath    string
		wantEscaped string
		wantErr     bool
	}{
		{"http://host/x/%2e%2e/admin", "/admin", "/admin", false},
		{"http://host//admin/", "/admin/", "/admin/", false},
		{"http://host/a%2Fb", "/a/b", "/a%2Fb", false},
		{"http://host/x//a%2Fb", "/x/a/b", "/x/a%2Fb", false},
		{"http://host/x//%61dmin", "/x/admin", "/x/%61dmin", false},
		{"http://host/a%20b", "/a b", "/a%20b", false},
		{"http://host", "", "", false},
		{"http://host/..", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(subT *testing.T) {
			u, err := url.Parse(tt.in)
			if err != nil {
				subT.Fatal(err)
			}
			err = NormalizeURLPath(u)
			if (err != nil) != tt.wantErr {
				subT.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			if u.Path != tt.wantPath {
				subT.Errorf("Path: want %q, got %q", tt.wantPath, u.Path)
			}
			if got := u.EscapedPath(); got != tt.wantEscaped {
				subT.Errorf("EscapedPath: want %q, got %q", tt.wantEscaped, got)
			}
		})
	}
}

func TestHasEmptyOrDotSegment(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"/", false},
		{"*", false},
		{"/a/b", false},
		{"/a/b/", false},
		{"/a//b", true},
		{"//", true},
		{"/a/./b", true},
		{"/a/../b", true},
		{"/a/.", true},
		{"/a/..", true},
		{`/a\..\b`, false},
	}

	for _, tt := range tests {
		if got := HasEmptyOrDotSegment(tt.in); got != tt.want {
			t.Errorf("%q: want %v, got %v", tt.in, tt.want, got)
		}
	}
}

func FuzzNormalizePath(f *testing.F) {
	for _, seed := range []string{"", "/", "/a/b/../c", "/%2e%2e/a", "/a//b/./c/", "/a%2Fb", "/a%2F..%2Fb", "/.%2e/a", "%2", "/%", "/a/%2e%2e%2e"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, in string) {
		out, err := NormalizePath("/" + in)
		if err != nil {
			return
		}
		again, err := NormalizePath(out)
		if err != nil || again != out {
			t.Fatalf("not idempotent: %q -> %q -> %q, %v", in, out, again, err)
		}
		if HasEmptyOrDotSegment(out) {
			t.Fatalf("empty or dot segment left in %q from %q", out, in)
		}
		for _, segment := range strings.Split(out, "/") {
			if dotSegment(segment) > 0 {
				t.Fatalf("encoded dot segment left in %q from %q", out, in)
			}
		}
	})
}
