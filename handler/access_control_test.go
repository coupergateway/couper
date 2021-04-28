package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/errors"
)

func TestAccessControl_ServeHTTP(t *testing.T) {
	type fields struct {
		ac        ac.List
		protected http.Handler
	}

	newReq := func(method, target string) func(headerKey string, headerValue string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		return func(key, value string) *http.Request {
			req.Header.Set(key, value)
			return req
		}
	}

	newBasicAuth := func(user, pass string) *ac.BasicAuth {
		ba, err := ac.NewBasicAuth("ba-test", user, pass, "")
		if err != nil {
			t.Fatal(err)
		}
		return ba
	}

	defaultErrHandler := NewErrorHandler(nil, errors.DefaultJSON)

	tests := []struct {
		name           string
		fields         fields
		req            *http.Request
		expectedStatus int
	}{
		{"no access control", fields{nil, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusNoContent},

		{"with access control valid req", fields{ac.List{ac.NewItem("", ac.ValidateFunc(func(r *http.Request) error {
			return nil // valid
		}), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusNoContent},

		{"with access control invalid req/empty token", fields{ac.List{ac.NewItem("", ac.ValidateFunc(func(r *http.Request) error {
			return errors.Types["jwt"].Status(http.StatusUnauthorized)
		}), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusUnauthorized},

		{"with access control invalid req", fields{ac.List{ac.NewItem("", ac.ValidateFunc(func(r *http.Request) error {
			return fmt.Errorf("no! ")
		}), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusForbidden},

		{"basic_auth", fields{ac.List{ac.NewItem("", newBasicAuth("hans", ""), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", "Basic aGFuczovVnFoV3FsS1VrSVNzUC8K"), http.StatusUnauthorized},

		{"basic_auth /wo authorization header", fields{ac.List{ac.NewItem("", newBasicAuth("hans", ""), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", ""), http.StatusUnauthorized},

		{"basic_auth /w realm", fields{ac.List{ac.NewItem("", newBasicAuth("hans", ""), defaultErrHandler)}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", "Basic aGFuczovVnFoV3FsS1VrSVNzUC8K"), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAccessControl(tt.fields.protected, tt.fields.ac)

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
