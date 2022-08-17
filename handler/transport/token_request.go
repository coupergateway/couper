package transport

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
)

type TokenRequest struct {
	config      *config.TokenRequest
	getMu       sync.Mutex
	memStore    *cache.MemoryStore
	reqProducer producer.Roundtrip
	storageKey  string
}

func NewTokenRequest(conf *config.TokenRequest, memStore *cache.MemoryStore, reqProducer producer.Roundtrip) (RequestAuthorizer, error) {
	tr := &TokenRequest{
		config:      conf,
		memStore:    memStore,
		reqProducer: reqProducer,
	}
	tr.storageKey = fmt.Sprintf("TokenRequest-%p", tr)
	return tr, nil
}

func (t *TokenRequest) GetToken(req *http.Request) error {
	token := t.readToken()
	if token != "" {
		return nil
	}

	// block during read/request process
	t.getMu.Lock()
	defer t.getMu.Unlock()

	token = t.readToken()
	if token != "" {
		return nil
	}

	var (
		ttl int64
		err error
	)
	token, ttl, err = t.requestToken(req)
	if err != nil {
		return errors.Backend.Label(t.config.BackendName).Message("token request error").With(err)
	}

	t.memStore.Set(t.storageKey, token, ttl)
	return nil
}

func (t *TokenRequest) RetryWithToken(_ *http.Request, _ *http.Response) (bool, error) {
	return false, nil
}

func (t *TokenRequest) readToken() string {
	if data := t.memStore.Get(t.storageKey); data != nil {
		return data.(string)
	}

	return ""
}

func (t *TokenRequest) requestToken(req *http.Request) (string, int64, error) {
	results := make(chan *producer.Result, 1)
	ctx := context.WithValue(req.Context(), request.Wildcard, nil)           // disable handling this
	ctx = context.WithValue(ctx, request.BufferOptions, eval.BufferResponse) // always read out a possible token
	ctx = context.WithValue(ctx, request.TokenRequest, t.config.Name)        // set the name for variable mapping purposes
	outreq, _ := http.NewRequestWithContext(ctx, req.Method, "", nil)
	t.reqProducer.Produce(outreq, results)
	result := <-results
	if result.Err != nil {
		return "", 0, result.Err
	}

	trConf := &config.TokenRequest{Remain: t.config.Remain}
	bodyContent, _, diags := t.config.Remain.PartialContent(trConf.Schema(true))
	if diags.HasErrors() {
		return "", 0, diags
	}

	// obtain synced and already read beresp value; map to context variables
	hclCtx := eval.ContextFromRequest(req).HCLContextSync()
	eval.MapTokenResponse(hclCtx, t.config.Name)

	tokenVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "token")
	if err != nil {
		return "", 0, err
	}
	if tokenVal.IsNull() {
		return "", 0, errors.Backend.Label(t.config.BackendName).Message("token expression evaluates to null")
	}
	if tokenVal.Type() != cty.String {
		return "", 0, errors.Backend.Label(t.config.BackendName).Message("token expression must evaluate to a string")
	}

	ttlVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "ttl")
	if err != nil {
		return "", 0, err
	}
	if ttlVal.IsNull() {
		return "", 0, errors.Backend.Label(t.config.BackendName).Message("ttl expression evaluates to null")
	}
	if ttlVal.Type() != cty.String {
		return "", 0, errors.Backend.Label(t.config.BackendName).Message("ttl expression must evaluate to a string")
	}

	token := tokenVal.AsString()
	ttl := ttlVal.AsString()
	dur, parseErr := time.ParseDuration(ttl)
	if parseErr != nil {
		return "", 0, errors.Backend.Label(t.config.BackendName).Message("parsing ttl").With(parseErr)
	}

	return token, int64(dur.Seconds()), nil
}

func (t *TokenRequest) value() (string, string) {
	token := t.readToken()
	return t.config.Name, token
}
