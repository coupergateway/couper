package errors_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server/writer"
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
		t1.Run(tt.name, func(subT *testing.T) {
			rec := writer.NewResponseWriter(httptest.NewRecorder(), "")
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			errors.DefaultJSON.WithError(tt.err).ServeHTTP(rec, req)

			rec.Flush()

			if rec.StatusCode() != tt.expStatus {
				subT.Errorf("expected status %d, got: %d", tt.expStatus, rec.StatusCode())
			}
		})
	}
}
