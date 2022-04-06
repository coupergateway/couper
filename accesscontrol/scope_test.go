package accesscontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

func Test_requiredScope(t *testing.T) {
	tests := []struct {
		name string
		base map[string]string
		sm   map[string]string
		want map[string][]string
	}{
		{
			"only default no scope",
			map[string]string{},
			map[string]string{"*": ""},
			map[string][]string{"CONNECT": {}, "DELETE": {}, "GET": {}, "HEAD": {}, "OPTIONS": {}, "PATCH": {}, "POST": {}, "PUT": {}, "TRACE": {}},
		},
		{
			"only default read",
			map[string]string{},
			map[string]string{"*": "read"},
			map[string][]string{"CONNECT": {"read"}, "DELETE": {"read"}, "GET": {"read"}, "HEAD": {"read"}, "OPTIONS": {"read"}, "PATCH": {"read"}, "POST": {"read"}, "PUT": {"read"}, "TRACE": {"read"}},
		},
		{
			"simple scope, simple no scope",
			map[string]string{},
			map[string]string{"POST": "write", "PUT": ""},
			map[string][]string{"POST": {"write"}, "PUT": {}},
		},
		{
			"simple scope, simple no scope, with default",
			map[string]string{},
			map[string]string{"POST": "write", "PUT": "", "*": "read"},
			map[string][]string{"CONNECT": {"read"}, "DELETE": {"read"}, "GET": {"read"}, "HEAD": {"read"}, "OPTIONS": {"read"}, "PATCH": {"read"}, "POST": {"write"}, "PUT": {}, "TRACE": {"read"}},
		},
		{
			"default / simple scope, simple no scope",
			map[string]string{"*": "read"},
			map[string]string{"POST": "write", "PUT": ""},
			map[string][]string{"POST": {"read", "write"}, "PUT": {"read"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			rs := newRequiredScope()
			rs.addScopeMap(tt.base)
			rs.addScopeMap(tt.sm)
			if len(rs.scopes) != len(tt.want) {
				subT.Errorf("unexpected scopes: %#v, want: %#v", rs.scopes, tt.want)
				return
			}
			for op, wantScopes := range tt.want {
				scopes, exists := rs.scopes[op]
				if !exists {
					subT.Errorf("no scopes for operation %q", op)
					return
				}
				if len(scopes) != len(wantScopes) {
					subT.Errorf("unexpected scopes for %q: %#v, want: %#v", op, scopes, wantScopes)
					return
				}
				for i, s := range wantScopes {
					if scopes[i] != s {
						subT.Errorf("unexpected scopes for %q: %#v, want: %#v", op, scopes, wantScopes)
						return
					}
				}
			}
		})
	}
}

func Test_ScopeControl(t *testing.T) {
	tests := []struct {
		name            string
		permissionMaps  []map[string]string
		method          string
		grantedScope    []string
		wantErrorString string
	}{
		{
			"no method restrictions, no scope required, no scope granted",
			[]map[string]string{},
			http.MethodGet,
			nil,
			"",
		},
		{
			"method permitted, no scope required, no scope granted",
			[]map[string]string{{http.MethodGet: ""}},
			http.MethodGet,
			nil,
			"",
		},
		{
			"method permitted, scope required, scope granted",
			[]map[string]string{{http.MethodGet: "read"}},
			http.MethodGet,
			[]string{"read"},
			"",
		},
		{
			"method permitted, scope required, scopes granted",
			[]map[string]string{{http.MethodPost: "write"}},
			http.MethodPost,
			[]string{"read", "write"},
			"",
		},
		{
			"method permitted, scopes required, scopes granted",
			[]map[string]string{{"*": "read"}, {http.MethodPost: "write"}},
			http.MethodPost,
			[]string{"read", "write"},
			"",
		},
		{
			"all methods permitted, scope required, scope granted",
			[]map[string]string{{"*": "read"}},
			http.MethodPost,
			[]string{"read"},
			"",
		},
		{
			"method not permitted",
			[]map[string]string{{http.MethodGet: ""}},
			http.MethodPost,
			nil,
			"access control error: operation POST not permitted",
		},
		{
			"method permitted, scope required, no scope granted",
			[]map[string]string{{http.MethodGet: "read"}},
			http.MethodGet,
			nil,
			"access control error: no scope granted",
		},
		{
			"method permitted, scope required, wrong scope granted",
			[]map[string]string{{http.MethodPost: "write"}},
			http.MethodPost,
			[]string{"read"},
			`access control error: required scope "write" not granted`,
		},
		{
			"method permitted, scopes required, missing granted scope",
			[]map[string]string{{"*": "read"}, {http.MethodPost: "write"}},
			http.MethodPost,
			[]string{"read"},
			`access control error: required scope "write" not granted`,
		},
		{
			"method permitted, scopes required, missing granted scopes",
			[]map[string]string{{"*": "read"}, {http.MethodPost: "write"}},
			http.MethodPost,
			[]string{"foo"},
			`access control error: required scope "read" not granted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			sc := NewScopeControl(tt.permissionMaps)
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
