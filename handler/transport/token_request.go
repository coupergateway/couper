package transport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

type TokenRequest struct {
	authorizedBackendName string
	backend               http.RoundTripper
	config                *config.TokenRequest
	locks                 sync.Map
	memStore              *cache.MemoryStore
	storageKey            string
}

func NewTokenRequest(conf *config.TokenRequest, memStore *cache.MemoryStore, backend http.RoundTripper, authorizedBackendName string) (RequestAuthorizer, error) {
	tr := &TokenRequest{
		authorizedBackendName: authorizedBackendName,
		backend:               backend,
		config:                conf,
		memStore:              memStore,
		locks:                 sync.Map{},
	}
	tr.storageKey = fmt.Sprintf("TokenRequest-%p", tr)
	return tr, nil
}

func (t *TokenRequest) WithToken(req *http.Request) error {
	ctx := eval.ContextFromRequest(req)
	if token, terr := t.readToken(); terr != nil {
		return errors.Backend.Label(t.config.BackendName).Message("token read error").With(terr)
	} else if token != "" {
		ctx.WithBackendToken(t.authorizedBackendName, t.config.Name, token)
		return nil
	}

	value, _ := t.locks.LoadOrStore(t.storageKey, &sync.Mutex{})
	mutex := value.(*sync.Mutex)

	mutex.Lock()
	token, terr := t.readToken()
	if terr != nil {
		mutex.Unlock()
		return errors.Backend.Label(t.config.BackendName).Message("token read error").With(terr)
	} else if token != "" {
		mutex.Unlock()
		ctx.WithBackendToken(t.authorizedBackendName, t.config.Name, token)
		return nil
	}

	token, ttl, err := t.requestToken(ctx)
	if err != nil {
		mutex.Unlock()
		return errors.Backend.Label(t.config.BackendName).Message("token request error").With(err)
	}

	t.updateToken(token, ttl)
	mutex.Unlock()

	ctx.WithBackendToken(t.authorizedBackendName, t.config.Name, token)
	return nil
}

func (t *TokenRequest) RetryWithToken(req *http.Request, res *http.Response) (bool, error) {
	return false, nil
}

func (t *TokenRequest) readToken() (string, error) {
	if data := t.memStore.Get(t.storageKey); data != nil {
		return data.(string), nil
	}

	return "", nil
}

func (t *TokenRequest) requestToken(ctx *eval.Context) (string, int64, error) {
	hclCtx := ctx.HCLContext()
	bodyContent, _, diags := t.config.Remain.PartialContent(config.TokenRequest{Remain: t.config.Remain}.Schema(true))
	if diags.HasErrors() {
		return "", 0, diags
	}

	methodVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "method")
	if err != nil {
		return "", 0, err
	}
	method := seetie.ValueToString(methodVal)

	urlVal, err := eval.ValueFromAttribute(hclCtx, bodyContent, "url")
	if err != nil {
		return "", 0, err
	}
	url := seetie.ValueToString(urlVal)

	body, defaultContentType, err := eval.GetBody(hclCtx, bodyContent)
	if err != nil {
		return "", 0, err
	}

	if method == "" {
		method = http.MethodGet

		if len(body) > 0 {
			method = http.MethodPost
		}
	}

	outreq, err := http.NewRequest(strings.ToUpper(method), url, nil)
	if err != nil {
		return "", 0, err
	}

	if defaultContentType != "" {
		outreq.Header.Set("Content-Type", defaultContentType)
	}

	eval.SetBody(outreq, []byte(body))

	err = eval.ApplyRequestContext(hclCtx, t.config.Remain, outreq)
	if err != nil {
		return "", 0, err
	}

	outCtx := context.WithValue(outreq.Context(), request.BufferOptions, eval.BufferResponse)
	outreq = outreq.WithContext(outCtx)
	resp, err := t.backend.RoundTrip(outreq)
	if err != nil {
		return "", 0, err
	}

	ctx = ctx.WithTokenresp(resp, true)
	hclCtx = ctx.HCLContext()

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

func (t *TokenRequest) updateToken(token string, ttl int64) {
	if t.memStore != nil {
		t.memStore.Set(t.storageKey, token, ttl)
	}
}
