package transport

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
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
		ctx.WithBackendToken(t.authorizedBackendName, token)
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
		ctx.WithBackendToken(t.authorizedBackendName, token)
		return nil
	}

	token, ttl, err := t.requestToken()
	if err != nil {
		mutex.Unlock()
		return errors.Backend.Label(t.config.BackendName).Message("token request error").With(err)
	}

	t.updateToken(token, ttl)
	mutex.Unlock()

	ctx.WithBackendToken(t.authorizedBackendName, token)
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

func (t *TokenRequest) requestToken() (string, int64, error) {
	// TODO:
	// * create token request from t.config
	// * RoundTrip on t.backend with token request
	// * store response in top-level variable token_response
	// * evaluate token expression
	// * evaluate ttl expression
	return "my_token", 10, nil
}

func (t *TokenRequest) updateToken(token string, ttl int64) {
	if t.memStore != nil {
		t.memStore.Set(t.storageKey, token, ttl)
	}
}
