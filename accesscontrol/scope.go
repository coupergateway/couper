package accesscontrol

import (
	"net/http"
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
