package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/access_control"
	"github.com/avenga/couper/errors"
)

func TestAccessControl_ServeHTTP(t *testing.T) {
	type fields struct {
		ac        access_control.List
		protected http.Handler
	}

	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		expectedStatus int
	}{
		{"no access control", fields{nil, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusNoContent},
		{"with access control valid req", fields{access_control.List{access_control.ValidateFunc(func(r *http.Request) error {
			return nil // valid
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusNoContent},
		{"with access control invalid req/empty token", fields{access_control.List{access_control.ValidateFunc(func(r *http.Request) error {
			return access_control.ErrorEmptyToken
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusUnauthorized},
		{"with access control invalid req", fields{access_control.List{access_control.ValidateFunc(func(r *http.Request) error {
			return fmt.Errorf("no! ")
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAccessControl(tt.fields.protected, errors.DefaultJSON, tt.fields.ac...)

			res := httptest.NewRecorder()
			a.ServeHTTP(res, tt.req)

			if !res.Flushed {
				res.Flush()
			}

			if res.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got: %d", tt.expectedStatus, res.Code)
			}
		})
	}
}
