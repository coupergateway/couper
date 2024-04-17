package utils_test

import (
	"testing"

	"github.com/coupergateway/couper/utils"
)

func TestUtils_JoinPath(t *testing.T) {
	if p := utils.JoinPath(); p != "" {
		t.Errorf("Unexpected path %q given, expected %q", p, "")
	}
	if p := utils.JoinPath("/", "", ""); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("", "/", ""); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("", "", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("/", "/", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinPath("/", "/foo", "/"); p != "/foo/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/")
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
	if p := utils.JoinPath("/foo", "/./."); p != "/foo" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo")
	}
	if p := utils.JoinPath("/foo", "/.."); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
}

func TestUtils_JoinOpenAPIPath(t *testing.T) {
	if p := utils.JoinOpenAPIPath(); p != "" {
		t.Errorf("Unexpected path %q given, expected %q", p, "")
	}
	if p := utils.JoinOpenAPIPath("/", "", ""); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinOpenAPIPath("", "/", ""); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinOpenAPIPath("", "", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinOpenAPIPath("/", "/", "/"); p != "/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/")
	}
	if p := utils.JoinOpenAPIPath("/", "/foo", "/"); p != "/foo" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo")
	}
	if p := utils.JoinOpenAPIPath("/", "/", "//"); p != "//" {
		t.Errorf("Unexpected path %q given, expected %q", p, "//")
	}
	if p := utils.JoinOpenAPIPath("/foo", "/bar"); p != "/foo/bar" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar")
	}
	if p := utils.JoinOpenAPIPath("/foo", "/bar/"); p != "/foo/bar/" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar/")
	}
	if p := utils.JoinOpenAPIPath("/foo/", "/bar//"); p != "/foo/bar//" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/bar//")
	}
	if p := utils.JoinOpenAPIPath("/foo", "/./."); p != "/foo/./." {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/./.")
	}
	if p := utils.JoinOpenAPIPath("/foo/", "/../"); p != "/foo/../" {
		t.Errorf("Unexpected path %q given, expected %q", p, "/foo/../")
	}
}
