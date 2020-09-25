package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/errors"
)

func TestAccessControl_ServeHTTP(t *testing.T) {
	type fields struct {
		ac        accesscontrol.List
		protected http.Handler
	}

	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		expectedStatus int
		wwwAuth        string
	}{
		{"no access control", fields{nil, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusNoContent, ""},

		{"with access control valid req", fields{accesscontrol.List{accesscontrol.ValidateFunc(func(r *http.Request) error {
			return nil // valid
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusNoContent, ""},

		{"with access control invalid req/empty token", fields{accesscontrol.List{accesscontrol.ValidateFunc(func(r *http.Request) error {
			return accesscontrol.ErrorEmptyToken
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusUnauthorized, ""},

		{"with access control invalid req", fields{accesscontrol.List{accesscontrol.ValidateFunc(func(r *http.Request) error {
			return fmt.Errorf("no! ")
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusForbidden, ""},

		{"with basic_auth", fields{accesscontrol.List{accesscontrol.ValidateFunc(func(r *http.Request) error {
			return accesscontrol.NewBasicAuthUnauthorizedError("")
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusUnauthorized, "Basic"},

		{"with basic_auth with realm", fields{accesscontrol.List{accesscontrol.ValidateFunc(func(r *http.Request) error {
			return accesscontrol.NewBasicAuthUnauthorizedError("My-Realn")
		})}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, httptest.NewRequest("GET", "http://ac.test/", nil), http.StatusUnauthorized, "Basic realm=My-Realn"},
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

			if res.Header().Get("WWW-Authenticate") != tt.wwwAuth {
				t.Errorf("Expected WWW-Auth header %q, got: %q", tt.wwwAuth, res.Header().Get("WWW-Authenticate"))
			}
		})
	}
}
