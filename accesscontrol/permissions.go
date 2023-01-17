package accesscontrol

import (
	"context"
	"net/http"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/internal/seetie"
)

type requiredPermissions struct {
	permission  string
	permissions map[string]string // permission per method
}

func newRequiredPermissions(permission string, permissionMap map[string]string) requiredPermissions {
	rp := requiredPermissions{}
	if permissionMap == nil {
		rp.permission = permission
		return rp
	}

	rp.setPermissionMap(permissionMap)
	return rp
}

func (r *requiredPermissions) setPermissionMap(permissionMap map[string]string) {
	r.permissions = make(map[string]string)
	if otherPermission, otherMethodExists := permissionMap["*"]; otherMethodExists {
		for _, method := range middleware.DefaultEndpointAllowedMethods {
			r.permissions[method] = otherPermission
		}
	}
	for method, permission := range permissionMap {
		if method == "*" {
			continue
		}
		r.permissions[method] = permission
	}
}

func (r *requiredPermissions) getPermission(method string) (string, error) {
	if r.permissions == nil {
		return r.permission, nil
	}

	permission, exists := r.permissions[method]
	if !exists {
		return "", errors.MethodNotAllowed.Messagef("method %s not allowed by required_permission", method)
	}
	return permission, nil
}

var _ AccessControl = &PermissionsControl{}

type PermissionsControl struct {
	permissionExpr hcl.Expression
}

func NewPermissionsControl(permissionExpr hcl.Expression) *PermissionsControl {
	return &PermissionsControl{permissionExpr: permissionExpr}
}

// Validate validates the granted permissions provided by access controls against the required permission.
func (p *PermissionsControl) Validate(req *http.Request) error {
	if p.permissionExpr == nil {
		return nil
	}

	permissionVal, err := eval.Value(eval.ContextFromRequest(req).HCLContext(), p.permissionExpr)
	if err != nil {
		return errors.Evaluation.With(err)
	}

	permission, permissionMap, err := seetie.ValueToPermission(permissionVal)
	if err != nil {
		return errors.Evaluation.With(err)
	}

	rp := newRequiredPermissions(permission, permissionMap)
	requiredPermission, err := rp.getPermission(req.Method)
	if err != nil {
		return err
	}

	if requiredPermission == "" {
		return nil
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, request.RequiredPermission, requiredPermission)
	*req = *req.WithContext(ctx)

	evalCtx := eval.ContextFromRequest(req)
	*req = *req.WithContext(evalCtx.WithClientRequest(req))

	grantedPermission, ok := ctx.Value(request.GrantedPermissions).([]string)
	if !ok {
		return errors.InsufficientPermissions.Messagef("no permissions granted")
	}
	if !hasGrantedPermission(grantedPermission, requiredPermission) {
		return errors.InsufficientPermissions.Messagef("required permission %q not granted", requiredPermission)
	}
	return nil
}

// hasGrantedPermission checks whether a given permission is in the granted permissions
func hasGrantedPermission(grantedPermissions []string, permission string) bool {
	for _, gp := range grantedPermissions {
		if gp == permission {
			return true
		}
	}
	return false
}
