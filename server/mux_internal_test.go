package server

import "testing"

func TestMux_JoinPath(t *testing.T) {
	if p := joinPath("/", "/", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := joinPath("/foo", "/bar"); p != "/foo/bar" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar")
	}
	if p := joinPath("/foo", "/bar/"); p != "/foo/bar/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar/")
	}
	if p := joinPath("/foo/", "/bar/"); p != "/foo/bar/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar/")
	}
}
