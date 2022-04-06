package accesscontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

func Test_requiredPermissions(t *testing.T) {
	tests := []struct {
		name string
		rpm  map[string]string
		want map[string]string
	}{
		{
			"only default no permission",
			map[string]string{"*": ""},
			map[string]string{"CONNECT": "", "DELETE": "", "GET": "", "HEAD": "", "OPTIONS": "", "PATCH": "", "POST": "", "PUT": "", "TRACE": ""},
		},
		{
			"only default read",
			map[string]string{"*": "read"},
			map[string]string{"CONNECT": "read", "DELETE": "read", "GET": "read", "HEAD": "read", "OPTIONS": "read", "PATCH": "read", "POST": "read", "PUT": "read", "TRACE": "read"},
		},
		{
			"simple permission, simple no permission",
			map[string]string{"POST": "write", "PUT": ""},
			map[string]string{"POST": "write", "PUT": ""},
		},
		{
			"simple permission, simple no permission, with default",
			map[string]string{"POST": "write", "PUT": "", "*": "read"},
			map[string]string{"CONNECT": "read", "DELETE": "read", "GET": "read", "HEAD": "read", "OPTIONS": "read", "PATCH": "read", "POST": "write", "PUT": "", "TRACE": "read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			rp := newRequiredPermissions()
			rp.setPermissionMap(tt.rpm)
			if len(rp.permissions) != len(tt.want) {
				subT.Errorf("unexpected permission: %#v, want: %#v", rp.permissions, tt.want)
				return
			}
			for method, wantPermission := range tt.want {
				permission, exists := rp.permissions[method]
				if !exists {
					subT.Errorf("no permission for method %q", method)
					return
				}
				if permission != wantPermission {
					subT.Errorf("unexpected permission for %q: %#v, want: %#v", method, permission, wantPermission)
					return
				}
			}
		})
	}
}

func Test_ScopeControl(t *testing.T) {
	tests := []struct {
		name            string
		permissionMap   map[string]string
		method          string
		grantedScope    []string
		wantErrorString string
	}{
		{
			"no method restrictions, no permission required, no scope granted",
			map[string]string{},
			http.MethodGet,
			nil,
			"",
		},
		{
			"method permitted, no permission required, no scope granted",
			map[string]string{http.MethodGet: ""},
			http.MethodGet,
			nil,
			"",
		},
		{
			"method permitted, permission required, scope granted",
			map[string]string{http.MethodGet: "read"},
			http.MethodGet,
			[]string{"read"},
			"",
		},
		{
			"method permitted, permission required, scopes granted",
			map[string]string{http.MethodPost: "write"},
			http.MethodPost,
			[]string{"read", "write"},
			"",
		},
		{
			"all methods permitted, permission required, scope granted",
			map[string]string{"*": "read"},
			http.MethodPost,
			[]string{"read"},
			"",
		},
		{
			"method not permitted",
			map[string]string{http.MethodGet: ""},
			http.MethodPost,
			nil,
			"access control error: method POST not permitted",
		},
		{
			"method permitted, permission required, no scope granted",
			map[string]string{http.MethodGet: "read"},
			http.MethodGet,
			nil,
			"access control error: no scope granted",
		},
		{
			"method permitted, permission required, wrong scope granted",
			map[string]string{http.MethodPost: "write"},
			http.MethodPost,
			[]string{"read"},
			`access control error: required permission "write" not granted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			sc := NewScopeControl(tt.permissionMap)
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.grantedScope != nil {
				ctx := req.Context()
				ctx = context.WithValue(ctx, request.Scopes, tt.grantedScope)
				*req = *req.WithContext(ctx)
			}
			err := sc.Validate(req)
			if tt.wantErrorString == "" && err == nil {
				return
			}
			if tt.wantErrorString == "" && err != nil {
				logErr := err.(errors.GoError)
				subT.Errorf("no error expected, was: %#q", logErr.LogError())
				return
			}
			if tt.wantErrorString != "" && err == nil {
				subT.Errorf("no error thrown, expected: %q", tt.wantErrorString)
				return
			}
			logErr := err.(errors.GoError)
			if tt.wantErrorString != logErr.LogError() {
				subT.Errorf("unexpected error thrown, expected: %q, was: %q", tt.wantErrorString, logErr.LogError())
				return
			}
		})
	}
}
