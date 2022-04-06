package accesscontrol

import (
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler/middleware"
)

type requiredPermissions struct {
	permissions map[string]string
}

func newRequiredPermissions() requiredPermissions {
	return requiredPermissions{permissions: make(map[string]string)}
}

func (r *requiredPermissions) setPermissionMap(permissionMap map[string]string) {
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

var _ AccessControl = &ScopeControl{}

type ScopeControl struct {
	required requiredPermissions
}

func NewScopeControl(permissionMap map[string]string) *ScopeControl {
	rp := newRequiredPermissions()
	if permissionMap != nil {
		rp.setPermissionMap(permissionMap)
	}
	return &ScopeControl{required: rp}
}

// Validate validates the scope values provided by access controls against the required permission.
func (s *ScopeControl) Validate(req *http.Request) error {
	if len(s.required.permissions) == 0 {
		return nil
	}
	requiredPermission, exists := s.required.permissions[req.Method]
	if !exists {
		return errors.MethodNotAllowed.Messagef("method %s not allowed by beta_required_permission", req.Method)
	}
	if requiredPermission == "" {
		return nil
	}
	ctx := req.Context()
	grantedScope, ok := ctx.Value(request.Scopes).([]string)
	if !ok {
		return errors.BetaInsufficientScope.Messagef("no scope granted")
	}
	if !hasGrantedScope(grantedScope, requiredPermission) {
		return errors.BetaInsufficientScope.Messagef("required permission %q not granted", requiredPermission)
	}
	return nil
}

func hasGrantedScope(grantedScope []string, permission string) bool {
	for _, gs := range grantedScope {
		if gs == permission {
			return true
		}
	}
	return false
}
