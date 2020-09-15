package handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/avenga/couper/handler"
)

func TestSpa_ServeHTTP(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name         string
		filePath     string
		req          *http.Request
		expectedCode int
	}{
		{"serve bootstrap file", "testdata/spa/app.html", httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK},
		{"serve no bootstrap file", "testdata/spa/not_exist.html", httptest.NewRequest(http.MethodGet, "/", nil), http.StatusNotFound},
		{"serve bootstrap dir", "testdata/spa", httptest.NewRequest(http.MethodGet, "/", nil), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := handler.NewSpa(path.Join(wd, tt.filePath))

			res := httptest.NewRecorder()
			s.ServeHTTP(res, tt.req)

			if !res.Flushed {
				res.Flush()
			}

			if res.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got: %d", tt.expectedCode, res.Code)
			}
		})
	}
}
