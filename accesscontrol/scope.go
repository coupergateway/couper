package accesscontrol

import (
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var supportedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

type requiredPermissions struct {
	permissions map[string][]string
}

func newRequiredPermissions() requiredPermissions {
	return requiredPermissions{permissions: make(map[string][]string)}
}

func (r *requiredPermissions) addPermissionMap(permissionMap map[string]string) {
	otherPermission, otherMethodExists := permissionMap["*"]
	for _, method := range supportedMethods {
		permission, exists := permissionMap[method]
		if exists {
			r.addPermissionForMethod(method, permission)
		} else if otherMethodExists {
			r.addPermissionForMethod(method, otherPermission)
		} else {
			delete(r.permissions, method)
		}
	}
}

func (r *requiredPermissions) addPermissionForMethod(method, permission string) {
	permissions, exists := r.permissions[method]
	if !exists {
		if permission == "" {
			// method permitted without permission
			r.permissions[method] = []string{}
			return
		}
		// method permitted with required permission
		r.permissions[method] = []string{permission}
		return
	}
	// no additional permission required
	if permission == "" {
		return
	}
	// add permission to required permissions for this method
	r.permissions[method] = append(permissions, permission)
}

var _ AccessControl = &ScopeControl{}

type ScopeControl struct {
	required requiredPermissions
}

func NewScopeControl(permissionMaps []map[string]string) *ScopeControl {
	rp := newRequiredPermissions()
	for _, permissionMap := range permissionMaps {
		if permissionMap != nil {
			rp.addPermissionMap(permissionMap)
		}
	}
	return &ScopeControl{required: rp}
}

// Validate validates the scope values provided by access controls against the required permission.
func (s *ScopeControl) Validate(req *http.Request) error {
	if len(s.required.permissions) == 0 {
		return nil
	}
	requiredPermissions, exists := s.required.permissions[req.Method]
	if !exists {
		return errors.BetaOperationDenied.Messagef("method %s not permitted", req.Method)
	}
	ctx := req.Context()
	grantedScope, ok := ctx.Value(request.Scopes).([]string)
	if !ok && len(requiredPermissions) > 0 {
		return errors.BetaInsufficientScope.Messagef("no scope granted")
	}
	for _, rp := range requiredPermissions {
		if !hasGrantedScope(grantedScope, rp) {
			return errors.BetaInsufficientScope.Messagef("required permission %q not granted", rp)
		}
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
