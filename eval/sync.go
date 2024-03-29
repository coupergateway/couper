package eval

import (
	"context"
	"net/http"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval/variables"
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

func (sv *SyncedVariables) SetResp(beresp *http.Response) {
	ctx := beresp.Request.Context()
	name, bereqV, berespV := newBerespValues(ctx, beresp)

	sv.Set(ctx, name, bereqV, berespV)
}

// Set finalized cty req/resp pair.
func (sv *SyncedVariables) Set(ctx context.Context, reqName string, bereqV, berespV cty.Value) {
	if tr, ok := ctx.Value(request.TokenRequest).(string); ok && tr != "" {
		reqName = TokenRequestPrefix + tr
	}

	backendName, _ := ctx.Value(request.BackendName).(string)

	sv.items.Store(reqName, &syncPair{
		backendName: backendName,
		bereq:       bereqV,
		beresp:      berespV,
		name:        reqName,
	})
}

func (sv *SyncedVariables) Sync(vars map[string]cty.Value) {
	var bereqs, beresps map[string]cty.Value
	if brs, ok := vars[variables.BackendRequests]; ok {
		bereqs = brs.AsValueMap()
	}

	if brps, ok := vars[variables.BackendResponses]; ok {
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

	vars[variables.BackendRequests] = cty.ObjectVal(bereqs)
	vars[variables.BackendResponses] = cty.ObjectVal(beresps)
}
