package accesscontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

func Test_requiredPermissions(t *testing.T) {
	tests := []struct {
		name string
		rp   string
		rpm  map[string]string
		want map[string]string
	}{
		{
			"no permission (string)",
			"",
			nil,
			nil,
		},
		{
			"no permission for default methods",
			"",
			map[string]string{"*": ""},
			map[string]string{"DELETE": "", "GET": "", "HEAD": "", "OPTIONS": "", "PATCH": "", "POST": "", "PUT": ""},
		},
		{
			"only default read",
			"",
			map[string]string{"*": "read"},
			map[string]string{"DELETE": "read", "GET": "read", "HEAD": "read", "OPTIONS": "read", "PATCH": "read", "POST": "read", "PUT": "read"},
		},
		{
			"simple permission, simple no permission",
			"",
			map[string]string{"POST": "write", "PUT": ""},
			map[string]string{"POST": "write", "PUT": ""},
		},
		{
			"simple permission, simple no permission, with default",
			"",
			map[string]string{"POST": "write", "PUT": "", "*": "read"},
			map[string]string{"DELETE": "read", "GET": "read", "HEAD": "read", "OPTIONS": "read", "PATCH": "read", "POST": "write", "PUT": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			rp := newRequiredPermissions(tt.rp, tt.rpm)
			if tt.want == nil {
				if rp.permissions != nil {
					subT.Errorf("expected permissions to be nil: %#v", rp.permissions)
				}
				return
			}
			if len(rp.permissions) != len(tt.want) {
				subT.Errorf("unexpected permission: %#v, want: %#v", rp.permissions, tt.want)
				return
			}
			for method, wantPermission := range tt.want {
				permission, err := rp.getPermission(method)
				if err != nil {
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

func Test_PermissionsControl(t *testing.T) {
	tests := []struct {
		name               string
		permission         string
		permissionMap      map[string]string
		method             string
		grantedPermissions []string
		wantErrorString    string
	}{
		{
			"no method restrictions, no permission required, no permission granted",
			"",
			nil,
			http.MethodGet,
			nil,
			"",
		},
		{
			"no method restrictions, no permission required, permission granted",
			"",
			nil,
			http.MethodGet,
			[]string{"read"},
			"",
		},
		{
			"no method permitted, no permission granted",
			"",
			map[string]string{},
			http.MethodGet,
			nil,
			"method not allowed error: method GET not allowed by required_permission",
		},
		{
			"no method permitted, permission granted",
			"",
			map[string]string{},
			http.MethodPost,
			[]string{"read"},
			"method not allowed error: method POST not allowed by required_permission",
		},
		{
			"method permitted, no permission required, no permission granted",
			"",
			map[string]string{http.MethodGet: ""},
			http.MethodGet,
			nil,
			"",
		},
		{
			"method permitted, permission required, permission granted",
			"",
			map[string]string{http.MethodGet: "read"},
			http.MethodGet,
			[]string{"read"},
			"",
		},
		{
			"method permitted, permission required, permission granted",
			"",
			map[string]string{http.MethodPost: "write"},
			http.MethodPost,
			[]string{"read", "write"},
			"",
		},
		{
			"permission required for all methods, permission granted",
			"read",
			nil,
			"BREW",
			[]string{"read"},
			"",
		},
		{
			"method permitted, permission required, permission granted",
			"",
			map[string]string{"Brew": "tea"},
			"BREW",
			[]string{"tea"},
			"",
		},
		{
			"method permitted, permission required, missing required permission",
			"",
			map[string]string{"Brew": "tea"},
			"BREW",
			[]string{"coffee"},
			`access control error: required permission "tea" not granted`,
		},
		{
			"default methods permitted, permission required, permission granted",
			"",
			map[string]string{"*": "read"},
			http.MethodPost,
			[]string{"read"},
			"",
		},
		{
			"default methods permitted, permission required, permission granted but non-default method not allowed",
			"",
			map[string]string{"*": "read"},
			"BREW",
			[]string{"read"},
			"method not allowed error: method BREW not allowed by required_permission",
		},
		{
			"standard method not allowed",
			"",
			map[string]string{http.MethodGet: ""},
			http.MethodPost,
			nil,
			"method not allowed error: method POST not allowed by required_permission",
		},
		{
			"method permitted, permission required, no permissions granted",
			"",
			map[string]string{http.MethodGet: "read"},
			http.MethodGet,
			nil,
			"access control error: no permissions granted",
		},
		{
			"method permitted, permission required, missing required permission",
			"",
			map[string]string{http.MethodPost: "write"},
			http.MethodPost,
			[]string{"read"},
			`access control error: required permission "write" not granted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			var expr hcl.Expression
			if tt.permission != "" {
				expr = &hclsyntax.TemplateExpr{
					Parts: []hclsyntax.Expression{
						&hclsyntax.LiteralValueExpr{
							Val: cty.StringVal(tt.permission),
						},
					},
				}
			} else if tt.permissionMap != nil {
				var items []hclsyntax.ObjectConsItem
				for k, v := range tt.permissionMap {
					item := hclsyntax.ObjectConsItem{
						KeyExpr: &hclsyntax.ObjectConsKeyExpr{
							Wrapped: &hclsyntax.TemplateExpr{
								Parts: []hclsyntax.Expression{
									&hclsyntax.LiteralValueExpr{
										Val: cty.StringVal(k),
									},
								},
							},
						},
						ValueExpr: &hclsyntax.TemplateExpr{
							Parts: []hclsyntax.Expression{
								&hclsyntax.LiteralValueExpr{
									Val: cty.StringVal(v),
								},
							},
						},
					}
					items = append(items, item)
				}
				expr = &hclsyntax.ObjectConsExpr{
					Items: items,
				}
			} else {
				expr = nil
			}
			pc := NewPermissionsControl(expr)
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.grantedPermissions != nil {
				ctx := req.Context()
				ctx = context.WithValue(ctx, request.GrantedPermissions, tt.grantedPermissions)
				*req = *req.WithContext(ctx)
			}
			err := pc.Validate(req)
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
