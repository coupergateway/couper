package accesscontrol

import (
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var supportedOperations = []string{
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

type requiredScope struct {
	scopes map[string][]string
}

func newRequiredScope() requiredScope {
	return requiredScope{scopes: make(map[string][]string)}
}

func (r *requiredScope) addScopeMap(permissionMap map[string]string) {
	otherScope, otherMethodExists := permissionMap["*"]
	for _, op := range supportedOperations {
		scope, exists := permissionMap[op]
		if exists {
			r.addScopeForOperation(op, scope)
		} else if otherMethodExists {
			r.addScopeForOperation(op, otherScope)
		} else {
			delete(r.scopes, op)
		}
	}
}

func (r *requiredScope) addScopeForOperation(operation, scope string) {
	scopes, exists := r.scopes[operation]
	if !exists {
		if scope == "" {
			// method permitted without scope
			r.scopes[operation] = []string{}
			return
		}
		// method permitted with required scope
		r.scopes[operation] = []string{scope}
		return
	}
	// no additional scope required
	if scope == "" {
		return
	}
	// add scope to required scopes for this operation
	r.scopes[operation] = append(scopes, scope)
}

var _ AccessControl = &ScopeControl{}

type ScopeControl struct {
	required requiredScope
}

func NewScopeControl(permissionMaps []map[string]string) *ScopeControl {
	rs := newRequiredScope()
	for _, permissionMap := range permissionMaps {
		if permissionMap != nil {
			rs.addScopeMap(permissionMap)
		}
	}
	return &ScopeControl{required: rs}
}

// Validate validates the scope values provided by access controls against the required scope values.
func (s *ScopeControl) Validate(req *http.Request) error {
	if len(s.required.scopes) == 0 {
		return nil
	}
	requiredScopes, exists := s.required.scopes[req.Method]
	if !exists {
		return errors.BetaOperationDenied.Messagef("operation %s not permitted", req.Method)
	}
	ctx := req.Context()
	grantedScope, ok := ctx.Value(request.Scopes).([]string)
	if !ok && len(requiredScopes) > 0 {
		return errors.BetaInsufficientScope.Messagef("no scope granted")
	}
	for _, rs := range requiredScopes {
		if !hasGrantedScope(grantedScope, rs) {
			return errors.BetaInsufficientScope.Messagef("required scope %q not granted", rs)
		}
	}
	return nil
}

func hasGrantedScope(grantedScope []string, scope string) bool {
	for _, gs := range grantedScope {
		if gs == scope {
			return true
		}
	}
	return false
}
