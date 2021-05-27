package errors_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server/writer"
)

func TestTemplate_ServeError(t1 *testing.T) {
	log, _ := test.NewLogger()
	errors.SetLogger(log.WithContext(context.Background()))

	tests := []struct {
		name      string
		err       error
		expStatus int
	}{
		{"error type with default status", errors.BasicAuth, http.StatusUnauthorized},
		{"error type without status code /w fallback", &errors.Error{}, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t2 *testing.T) {
			rec := writer.NewResponseWriter(httptest.NewRecorder(), "")
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			errors.DefaultJSON.ServeError(tt.err).ServeHTTP(rec, req)

			rec.Flush()

			if rec.StatusCode() != tt.expStatus {
				t2.Errorf("expected status %d, got: %d", tt.expStatus, rec.StatusCode())
			}
		})
	}
}
