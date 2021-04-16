package ac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/accesscontrol"
)

func TestAccessControl_ServeHTTP(t *testing.T) {
	type fields struct {
		ac        accesscontrol.List
		protected http.Handler
	}

	newReq := func(method, target string) func(headerKey string, headerValue string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		return func(key, value string) *http.Request {
			req.Header.Set(key, value)
			return req
		}
	}

	newBasicAuth := func(user, pass, realm string) *accesscontrol.BasicAuth {
		ba, err := accesscontrol.NewBasicAuth("ba-test", user, pass, "", realm)
		if err != nil {
			t.Fatal(err)
		}
		return ba
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
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusNoContent, ""},

		{"with access control valid req", fields{accesscontrol.List{{Func: accesscontrol.ValidateFunc(func(r *http.Request) error {
			return nil // valid
		}), Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusNoContent, ""},

		{"with access control invalid req/empty token", fields{accesscontrol.List{{Func: accesscontrol.ValidateFunc(func(r *http.Request) error {
			return accesscontrol.JWTError
		}), Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusUnauthorized, ""},

		{"with access control invalid req", fields{accesscontrol.List{{Func: accesscontrol.ValidateFunc(func(r *http.Request) error {
			return fmt.Errorf("no! ")
		}), Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusGone)
		})}, newReq("GET", "http://ac.test/")("", ""), http.StatusForbidden, ""},

		{"basic_auth", fields{accesscontrol.List{{Func: newBasicAuth("hans", "", ""),
			Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", "Basic aGFuczovVnFoV3FsS1VrSVNzUC8K"), http.StatusUnauthorized, "Basic"},

		{"basic_auth /wo authorization header", fields{accesscontrol.List{{Func: newBasicAuth("hans", "", ""),
			Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", ""), http.StatusUnauthorized, "Basic"},

		{"basic_auth /w realm", fields{accesscontrol.List{{Func: newBasicAuth("hans", "", "My-Realm"),
			Name: ""}}, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})}, newReq("GET", "http://ac.test/")("Authorization", "Basic aGFuczovVnFoV3FsS1VrSVNzUC8K"), http.StatusUnauthorized, "Basic realm=My-Realm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("TODO: error_handler")
			a := NewAccessControl(tt.fields.protected, nil, tt.fields.ac)

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
