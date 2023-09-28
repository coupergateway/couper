package eval

import (
	"net/http"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/request"
)

const TokenRequestPrefix = "_tr_"

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
	backendName   string
	bereq, beresp cty.Value
}

// Set finalized cty req/resp pair.
func (sv *SyncedVariables) Set(beresp *http.Response) {
	ctx := beresp.Request.Context()
	name, bereqV, berespV := newBerespValues(ctx, true, beresp)

	if tr, ok := ctx.Value(request.TokenRequest).(string); ok && tr != "" {
		name = TokenRequestPrefix + tr
	}

	backendName, _ := ctx.Value(request.BackendName).(string)

	sv.items.Store(name, &syncPair{
		backendName: backendName,
		bereq:       bereqV,
		beresp:      berespV,
		name:        name,
	})
}

func (sv *SyncedVariables) Sync(variables map[string]cty.Value) {
	var bereqs, beresps map[string]cty.Value
	if brs, ok := variables[BackendRequests]; ok {
		bereqs = brs.AsValueMap()
	}

	if brps, ok := variables[BackendResponses]; ok {
		beresps = brps.AsValueMap()
	}

	sv.items.Range(func(key, value interface{}) bool {
		p := value.(*syncPair)
		if bereqs == nil {
			bereqs = make(map[string]cty.Value)
		}
		name := key.(string)
		bereqs[name] = p.bereq

		if beresps == nil {
			beresps = make(map[string]cty.Value)
		}
		beresps[name] = p.beresp

		return true
	})

	variables[BackendRequests] = cty.ObjectVal(bereqs)
	variables[BackendResponses] = cty.ObjectVal(beresps)
}
