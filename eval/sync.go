package eval

import (
	"net/http"
	"sync"

	"github.com/zclconf/go-cty/cty"
)

type SyncedVariables struct {
	items sync.Map
}

func NewSyncedVariables() *SyncedVariables {
	return &SyncedVariables{
		items: sync.Map{},
	}
}

type syncPair struct {
	name          string
	bereq, beresp cty.Value
}

// Set finalized cty req/resp pair.
func (sv *SyncedVariables) Set(beresp *http.Response) {
	name, bereqV, berespV := newBerespValues(beresp.Request.Context(), true, beresp)
	sv.items.Store(name, &syncPair{
		name:   name,
		bereq:  bereqV,
		beresp: berespV,
	})
}

func (sv *SyncedVariables) Sync(variables map[string]cty.Value) {
	bereqs := variables[BackendRequests].AsValueMap()
	beresps := variables[BackendResponses].AsValueMap()

	sv.items.Range(func(key, value interface{}) bool {
		p := value.(*syncPair)
		if bereqs == nil {
			bereqs = make(map[string]cty.Value)
		}
		bereqs[p.name] = p.bereq

		if beresps == nil {
			beresps = make(map[string]cty.Value)
		}
		beresps[p.name] = p.beresp

		return true
	})

	variables[BackendRequests] = cty.ObjectVal(bereqs)
	variables[BackendResponses] = cty.ObjectVal(beresps)
}
