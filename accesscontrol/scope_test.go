package accesscontrol

import (
	"testing"
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
			map[string][]string{"CONNECT": []string{}, "DELETE": []string{}, "GET": []string{}, "HEAD": []string{}, "OPTIONS": []string{}, "PATCH": []string{}, "POST": []string{}, "PUT": []string{}, "TRACE": []string{}},
		},
		{
			"only default read",
			map[string]string{},
			map[string]string{"*": "read"},
			map[string][]string{"CONNECT": []string{"read"}, "DELETE": []string{"read"}, "GET": []string{"read"}, "HEAD": []string{"read"}, "OPTIONS": []string{"read"}, "PATCH": []string{"read"}, "POST": []string{"read"}, "PUT": []string{"read"}, "TRACE": []string{"read"}},
		},
		{
			"simple scope, simple no scope",
			map[string]string{},
			map[string]string{"POST": "write", "PUT": ""},
			map[string][]string{"POST": []string{"write"}, "PUT": []string{}},
		},
		{
			"simple scope, simple no scope, with default",
			map[string]string{},
			map[string]string{"POST": "write", "PUT": "", "*": "read"},
			map[string][]string{"CONNECT": []string{"read"}, "DELETE": []string{"read"}, "GET": []string{"read"}, "HEAD": []string{"read"}, "OPTIONS": []string{"read"}, "PATCH": []string{"read"}, "POST": []string{"write"}, "PUT": []string{}, "TRACE": []string{"read"}},
		},
		{
			"default / simple scope, simple no scope",
			map[string]string{"*": "read"},
			map[string]string{"POST": "write", "PUT": ""},
			map[string][]string{"POST": []string{"read", "write"}, "PUT": []string{"read"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			rs := newRequiredScope()
			rs.addScopeMap(tt.base)
			rs.addScopeMap(tt.sm)
			if len(rs.scopes) != len(tt.want) {
				t.Errorf("unexpected scopes: %#v, want: %#v", rs.scopes, tt.want)
				return
			}
			for op, wantScopes := range tt.want {
				scopes, exists := rs.scopes[op]
				if !exists {
					t.Errorf("no scopes for operation %q", op)
					return
				}
				if len(scopes) != len(wantScopes) {
					t.Errorf("unexpected scopes for %q: %#v, want: %#v", op, scopes, wantScopes)
					return
				}
				for i, s := range wantScopes {
					if scopes[i] != s {
						t.Errorf("unexpected scopes for %q: %#v, want: %#v", op, scopes, wantScopes)
						return
					}
				}
			}
		})
	}
}
