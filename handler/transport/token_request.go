package transport

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
)

var (
	_ RequestAuthorizer = &TokenRequest{}
)

type TokenRequest struct {
	config      *config.TokenRequest
	mu          sync.Mutex
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
	t.mu.Lock()
	defer t.mu.Unlock()

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
		return errors.Request.Label(t.config.Name).Message("token request failed").With(err)
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
	ctx := context.WithValue(req.Context(), request.Wildcard, nil)           // disable handling this
	ctx = context.WithValue(ctx, request.BufferOptions, eval.BufferResponse|eval.JSONParseResponse) // always read out a possible token
	ctx = context.WithValue(ctx, request.TokenRequest, t.config.Name)        // set the name for variable mapping purposes
	outreq, _ := http.NewRequestWithContext(ctx, req.Method, "", nil)
	result := t.reqProducer.Produce(outreq)
	if result.Err != nil {
		return "", 0, result.Err
	}

	// obtain synced and already read beresp value; map to context variables
	hclCtx := eval.ContextFromRequest(req).HCLContextSync()
	eval.MapTokenResponse(hclCtx, t.config.Name)

	tokenRequestBody := t.config.HCLBody()
	tokenVal, err := eval.ValueFromBodyAttribute(hclCtx, tokenRequestBody, "token")
	if err != nil {
		return "", 0, err
	}
	if tokenVal.IsNull() {
		return "", 0, fmt.Errorf("token expression evaluates to null")
	}
	if tokenVal.Type() != cty.String {
		return "", 0, fmt.Errorf("token expression must evaluate to a string")
	}

	ttlVal, err := eval.ValueFromBodyAttribute(hclCtx, tokenRequestBody, "ttl")
	if err != nil {
		return "", 0, err
	}
	if ttlVal.IsNull() {
		return "", 0, fmt.Errorf("ttl expression evaluates to null")
	}
	if ttlVal.Type() != cty.String {
		return "", 0, fmt.Errorf("ttl expression must evaluate to a string")
	}

	token := tokenVal.AsString()
	ttl := ttlVal.AsString()
	dur, parseErr := config.ParseDuration("ttl", ttl, 0)
	if parseErr != nil {
		return "", 0, parseErr
	}

	return token, int64(dur.Seconds()), nil
}

func (t *TokenRequest) value() (string, string) {
	token := t.readToken()
	return t.config.Name, token
}
