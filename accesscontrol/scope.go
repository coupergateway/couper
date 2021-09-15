package accesscontrol

import (
	"fmt"
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

func (r *requiredScope) addScopeMap(scopeMap map[string]string) {
	otherScope, otherMethodExists := scopeMap["*"]
	for _, op := range supportedOperations {
		scope, exists := scopeMap[op]
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

func NewScopeControl(scopeMaps []map[string]string) *ScopeControl {
	rs := newRequiredScope()
	for _, scopeMap := range scopeMaps {
		if scopeMap != nil {
			rs.addScopeMap(scopeMap)
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
		return errors.BetaOperationDenied.With(fmt.Errorf("operation %s not permitted", req.Method))
	}
	ctx := req.Context()
	grantedScope, ok := ctx.Value(request.Scopes).([]string)
	if !ok && len(requiredScopes) > 0 {
		return errors.BetaInsufficientScope.With(fmt.Errorf("no scope granted"))
	}
	for _, rs := range requiredScopes {
		if !hasGrantedScope(grantedScope, rs) {
			return errors.BetaInsufficientScope.With(fmt.Errorf("required scope %q not granted", rs))
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
