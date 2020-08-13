package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.avenga.cloud/couper/gateway/errors"
)

func TestFile_ServeHTTP(t *testing.T) {
	type fields struct {
		basePath   string
		errFile    string
		docRootDir string
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name         string
		fields       fields
		req          *http.Request
		expectedCode int
	}{
		{"not found", fields{}, httptest.NewRequest(http.MethodGet, "http://domain.test/", nil), http.StatusNotFound},
		{"index.html", fields{docRootDir: "testdata/file"}, httptest.NewRequest(http.MethodGet, "http://domain.test/", nil), http.StatusOK},
		{"serve file", fields{docRootDir: "testdata/file"}, httptest.NewRequest(http.MethodGet, "http://domain.test/favicon.ico", nil), http.StatusOK},
		{"not found within root dir", fields{docRootDir: "testdata/file"}, httptest.NewRequest(http.MethodGet, "http://domain.test/.git", nil), http.StatusNotFound},
		{"serve file /w basePath", fields{docRootDir: "testdata/file", basePath: "/base"}, httptest.NewRequest(http.MethodGet, "http://domain.test/base/favicon.ico", nil), http.StatusOK},
		{"not found within root dir /w basePath", fields{docRootDir: "testdata/file", basePath: "/base"}, httptest.NewRequest(http.MethodGet, "http://domain.test/base/.git", nil), http.StatusNotFound},
		{"not found /w errorFile", fields{errFile: "testdata/file_err_doc.html"}, httptest.NewRequest(http.MethodGet, "http://domain.test/", nil), http.StatusNotFound},
		{"not found /w errorFile HEAD", fields{errFile: "testdata/file_err_doc.html"}, httptest.NewRequest(http.MethodHead, "http://domain.test/", nil), http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFile(wd, tt.fields.basePath, tt.fields.docRootDir, errors.DefaultHTML)

			rec := httptest.NewRecorder()
			f.ServeHTTP(rec, tt.req)

			if !rec.Flushed {
				rec.Flush()
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got: %d", tt.expectedCode, rec.Code)
			}
		})
	}
}
