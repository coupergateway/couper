package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFile(wd, tt.fields.basePath, tt.fields.docRootDir, tt.fields.errFile)

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
