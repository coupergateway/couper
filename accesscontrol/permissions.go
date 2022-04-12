package accesscontrol

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/middleware"
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
		return "", errors.MethodNotAllowed.Messagef("method %s not allowed by beta_required_permission", method)
	}
	return permission, nil
}

var _ AccessControl = &PermissionsControl{}

type PermissionsControl struct {
	required requiredPermissions
}

func NewPermissionsControl(permission string, permissionMap map[string]string) *PermissionsControl {
	rp := newRequiredPermissions(permission, permissionMap)
	return &PermissionsControl{required: rp}
}

// Validate validates the granted permissions provided by access controls against the required permission.
func (p *PermissionsControl) Validate(req *http.Request) error {
	requiredPermission, err := p.required.getPermission(req.Method)
	if err != nil {
		return err
	}

	if requiredPermission == "" {
		return nil
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, request.BetaRequiredPermission, requiredPermission)
	*req = *req.WithContext(ctx)

	evalCtx := eval.ContextFromRequest(req)
	*req = *req.WithContext(evalCtx.WithClientRequest(req))

	grantedPermission, ok := ctx.Value(request.BetaGrantedPermissions).([]string)
	if !ok {
		return errors.BetaInsufficientPermissions.Messagef("no permissions granted")
	}
	if !hasGrantedPermission(grantedPermission, requiredPermission) {
		return errors.BetaInsufficientPermissions.Messagef("required permission %q not granted", requiredPermission)
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
