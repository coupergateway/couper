package server

import (
	"net/http/httptest"
	"testing"
)

func TestNormalizeRequestPath(t *testing.T) {
	tests := []struct {
		target      string
		wantPath    string
		wantEscaped string
		wantErr     bool
	}{
		{"/x/%2e%2e/admin", "/admin", "/admin", false},
		{"/x/../admin", "/admin", "/admin", false},
		{"//admin/", "/admin/", "/admin/", false},
		{"/a%2Fb", "/a/b", "/a%2Fb", false},
		{"/x//a%2Fb", "/x/a/b", "/x/a%2Fb", false},
		{"/a%2Fb/../c", "/c", "/c", false},
		{"/x//%61dmin", "/x/admin", "/x/%61dmin", false},
		{"/%61dmin", "/admin", "/%61dmin", false},
		{"/a%5C..%5Cb", `/a\..\b`, "/a%5C..%5Cb", false},
		{"/a%20b", "/a b", "/a%20b", false},
		{"/..", "", "", true},
		{"/%2e%2e/admin", "", "", true},
		{"/x%2F..%2Fadmin", "", "", true},
		{"/admin%2F..", "", "", true},
		{"/a%2F%2Fb", "", "", true},
		{"/a%2F.%2Fb", "", "", true},
		{"/x//y%2F..%2Fz", "", "", true},
		// A raw "|" must not switch the normalizer to the re-encoded path.
		{"/x%2F..%2Fadmin/panel.txt|", "", "", true},
		{"/admin%2F..%2Fpublic/file.txt|", "", "", true},
		{"/a%2F%2Fb|", "", "", true},
		{"/a|b/../c", "/c", "/c", false},
		{"/a|b/../c|", "/c|", "/c%7C", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(subT *testing.T) {
			req := httptest.NewRequest("GET", tt.target, nil)
			err := normalizeRequestPath(req)
			if (err != nil) != tt.wantErr {
				subT.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if err != nil {
				sent := httptest.NewRequest("GET", tt.target, nil).URL
				if req.URL.Path != sent.Path || req.URL.RawPath != sent.RawPath {
					subT.Errorf("URL changed on error: Path %q, RawPath %q", req.URL.Path, req.URL.RawPath)
				}
				return
			}
			if req.URL.Path != tt.wantPath {
				subT.Errorf("Path: want %q, got %q", tt.wantPath, req.URL.Path)
			}
			if got := req.URL.EscapedPath(); got != tt.wantEscaped {
				subT.Errorf("EscapedPath: want %q, got %q", tt.wantEscaped, got)
			}
			if got := req.URL.RequestURI(); got != tt.wantEscaped {
				subT.Errorf("RequestURI: want %q, got %q", tt.wantEscaped, got)
			}
		})
	}
}
