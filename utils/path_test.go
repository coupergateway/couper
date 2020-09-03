package utils_test

import (
	"testing"

	"github.com/avenga/couper/utils"
)

func TestUtils_JoinPath(t *testing.T) {
	if p := utils.JoinPath("/", "/", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("/", "/", "//"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("/foo", "/bar"); p != "/foo/bar" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar")
	}
	if p := utils.JoinPath("/foo", "/bar/"); p != "/foo/bar/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar/")
	}
	if p := utils.JoinPath("/foo/", "/bar/"); p != "/foo/bar/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar/")
	}
}
