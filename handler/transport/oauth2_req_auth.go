package transport

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/oauth2"
)

// OAuth2ReqAuth represents the transport <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	config       *config.OAuth2ReqAuth
	locks        sync.Map
	memStore     *cache.MemoryStore
	oauth2Client *oauth2.ClientCredentialsClient
	storageKey   string
}

// NewOAuth2ReqAuth implements the http.RoundTripper interface to wrap an existing Backend / http.RoundTripper
// to retrieve a valid token before passing the initial out request.
func NewOAuth2ReqAuth(conf *config.OAuth2ReqAuth, memStore *cache.MemoryStore,
	oauth2Client *oauth2.ClientCredentialsClient) TokenRequest {
	reqAuth := &OAuth2ReqAuth{
		config:       conf,
		oauth2Client: oauth2Client,
		memStore:     memStore,
		locks:        sync.Map{},
	}
	reqAuth.storageKey = fmt.Sprintf("oauth2-%p", reqAuth)
	return reqAuth
}

func (oa *OAuth2ReqAuth) WithToken(req *http.Request) error {
	if token, terr := oa.readAccessToken(); terr != nil {
		// TODO this error is not connected to the OAuth2 client's backend
		// In fact this can only be a JSON parse error or a missing access_token,
		// which will occur after having requested the token from the authorization
		// server. So the erroneous response will never be stored.
		return errors.Backend.Label(oa.config.BackendName).Message("token read error").With(terr)
	} else if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	value, _ := oa.locks.LoadOrStore(oa.storageKey, &sync.Mutex{})
	mutex := value.(*sync.Mutex)

	mutex.Lock()
	token, terr := oa.readAccessToken()
	if terr != nil {
		mutex.Unlock()
		return errors.Backend.Label(oa.config.BackendName).Message("token read error").With(terr)
	} else if token != "" {
		mutex.Unlock()
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	ctx := req.Context()
	tokenResponse, tokenResponseData, token, err := oa.oauth2Client.GetTokenResponse(ctx)
	if err != nil {
		mutex.Unlock()
		return errors.Backend.Label(oa.config.BackendName).Message("token request error").With(err)
	}

	oa.updateAccessToken(tokenResponse, tokenResponseData)
	mutex.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (oa *OAuth2ReqAuth) RetryWithToken(req *http.Request, res *http.Response) (bool, error) {
	if res == nil || res.StatusCode != http.StatusUnauthorized {
		return false, nil
	}

	oa.memStore.Del(oa.storageKey)

	ctx := req.Context()
	if retries, ok := ctx.Value(request.TokenRequestRetries).(uint8); !ok || retries < *oa.config.Retries {
		ctx = context.WithValue(ctx, request.TokenRequestRetries, retries+1)

		req.Header.Del("Authorization")
		err := oa.WithToken(req.WithContext(ctx)) // WithContext due to header manipulation
		return true, err
	}
	return false, nil
}

func (oa *OAuth2ReqAuth) readAccessToken() (string, error) {
	if data := oa.memStore.Get(oa.storageKey); data != nil {
		_, token, err := oauth2.ParseTokenResponse(data.([]byte))
		if err != nil {
			// err can only be JSON parse error, however non-JSON data should never be stored
			return "", err
		}

		return token, nil
	}

	return "", nil
}

func (oa *OAuth2ReqAuth) updateAccessToken(jsonBytes []byte, jData map[string]interface{}) {
	if oa.memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		oa.memStore.Set(oa.storageKey, jsonBytes, ttl)
	}
}
