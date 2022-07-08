package transport

import (
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
	oauth2Client *oauth2.Client
	storageKey   string
}

// NewOAuth2ReqAuth implements the http.RoundTripper interface to wrap an existing Backend / http.RoundTripper
// to retrieve a valid token before passing the initial out request.
func NewOAuth2ReqAuth(conf *config.OAuth2ReqAuth, memStore *cache.MemoryStore,
	asBackend http.RoundTripper) (TokenRequest, error) {

	if conf.GrantType != "client_credentials" && conf.GrantType != "password" {
		return nil, fmt.Errorf("grant_type %s not supported", conf.GrantType)
	}

	if conf.GrantType == "client_credentials" {
		// conf.Username undocumented feature!
		if conf.Username != "" {
			return nil, fmt.Errorf("username must not be set with grant_type=client_credentials")
		}
		// conf.Password undocumented feature!
		if conf.Password != "" {
			return nil, fmt.Errorf("password must not be set with grant_type=client_credentials")
		}
	}

	// grant_type password undocumented feature!
	// WARNING: this implementation is no proper password flow, but a flow with username and password to login _exactly one_ user
	// the received access token is stored in cache just like with the client credentials flow
	if conf.GrantType == "password" {
		if conf.Username == "" {
			return nil, fmt.Errorf("username must not be empty with grant_type=password")
		}
		if conf.Password == "" {
			return nil, fmt.Errorf("password must not be empty with grant_type=password")
		}
	}

	oauth2Client, err := oauth2.NewClient(conf.GrantType, conf, conf, asBackend)
	if err != nil {
		return nil, err
	}

	reqAuth := &OAuth2ReqAuth{
		config:       conf,
		oauth2Client: oauth2Client,
		memStore:     memStore,
		locks:        sync.Map{},
	}
	reqAuth.storageKey = fmt.Sprintf("oauth2-%p", reqAuth)
	return reqAuth, nil
}

func (oa *OAuth2ReqAuth) WithToken(req *http.Request) error {
	token := oa.readAccessToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	value, _ := oa.locks.LoadOrStore(oa.storageKey, &sync.Mutex{})
	mutex := value.(*sync.Mutex)

	mutex.Lock()
	token = oa.readAccessToken()
	if token != "" {
		mutex.Unlock()
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	requestParams := make(map[string]string)
	if oa.config.Scope != nil {
		requestParams["scope"] = *oa.config.Scope
	}
	// password and username undocumented feature!
	if oa.config.Password != "" || oa.config.Username != "" {
		requestParams["username"] = oa.config.Username
		requestParams["password"] = oa.config.Password
	}

	tokenResponseData, token, err := oa.oauth2Client.GetTokenResponse(req.Context(), requestParams)
	if err != nil {
		mutex.Unlock()
		return errors.Backend.Label(oa.config.BackendName).Message("token request error").With(err)
	}

	oa.updateAccessToken(token, tokenResponseData)
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
	if retries, ok := ctx.Value(request.TokenRequestRetries).(*uint8); !ok || *retries < *oa.config.Retries {
		*retries++ // increase ptr value instead of context value
		req.Header.Del("Authorization")
		err := oa.WithToken(req.WithContext(ctx)) // WithContext due to header manipulation
		return true, err
	}
	return false, nil
}

func (oa *OAuth2ReqAuth) readAccessToken() string {
	if data := oa.memStore.Get(oa.storageKey); data != nil {
		return data.(string)
	}

	return ""
}

func (oa *OAuth2ReqAuth) updateAccessToken(token string, jData map[string]interface{}) {
	if oa.memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		oa.memStore.Set(oa.storageKey, token, ttl)
	}
}
