package transport

import (
	"bytes"
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
	getMu                 sync.Mutex
	memStore              *cache.MemoryStore
	storageKey            string
}

func NewTokenRequest(conf *config.TokenRequest, memStore *cache.MemoryStore, backend http.RoundTripper, authorizedBackendName string) (RequestAuthorizer, error) {
	tr := &TokenRequest{
		authorizedBackendName: authorizedBackendName,
		backend:               backend,
		config:                conf,
		memStore:              memStore,
	}
	tr.storageKey = fmt.Sprintf("TokenRequest-%p", tr)
	return tr, nil
}

func (t *TokenRequest) GetToken(req *http.Request) error {
	if token, err := t.readToken(); err != nil {
		return errors.Backend.Label(t.config.BackendName).Message("token read error").With(err)
	} else if token != "" {
		return nil
	}

	// block during read/request process
	t.getMu.Lock()
	defer t.getMu.Unlock()

	token, terr := t.readToken()
	if terr != nil {
		return errors.Backend.Label(t.config.BackendName).Message("token read error").With(terr)
	} else if token != "" {
		return nil
	}

	ctx := eval.ContextFromRequest(req)
	token, ttl, err := t.requestToken(ctx)
	if err != nil {
		return errors.Backend.Label(t.config.BackendName).Message("token request error").With(err)
	}

	t.memStore.Set(t.storageKey, token, ttl)
	return nil
}

func (t *TokenRequest) RetryWithToken(_ *http.Request, _ *http.Response) (bool, error) {
	return false, nil
}

func (t *TokenRequest) readToken() (string, error) {
	if data := t.memStore.Get(t.storageKey); data != nil {
		return data.(string), nil
	}

	return "", nil
}

func (t *TokenRequest) requestToken(etx *eval.Context) (string, int64, error) {
	hclCtx := etx.HCLContextSync()

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

	outreq, err := http.NewRequest(strings.ToUpper(method), url, bytes.NewBufferString(body))
	if err != nil {
		return "", 0, err
	}

	if defaultContentType != "" {
		outreq.Header.Set("Content-Type", defaultContentType)
	}

	err = eval.ApplyRequestContext(hclCtx, t.config.Remain, outreq)
	if err != nil {
		return "", 0, err
	}

	// outCtx with "client" context due to syncedVars, cancel etc.
	outCtx := context.WithValue(etx, request.BufferOptions, eval.BufferResponse)
	outCtx = context.WithValue(outCtx, request.TokenRequest, t.config.Name)

	outreq = outreq.WithContext(outCtx)
	_, err = t.backend.RoundTrip(outreq)
	if err != nil {
		return "", 0, err
	}

	// obtain synced and already read beresp value; map to context variables
	hclCtx = etx.HCLContextSync()
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
	token, _ := t.readToken()
	return t.config.Name, token
}
