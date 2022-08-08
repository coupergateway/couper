package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
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
func NewOAuth2ReqAuth(evalCtx *hcl.EvalContext, conf *config.OAuth2ReqAuth, memStore *cache.MemoryStore,
	asBackend http.RoundTripper) (RequestAuthorizer, error) {

	if conf.GrantType != "client_credentials" && conf.GrantType != "password" && conf.GrantType != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		return nil, fmt.Errorf("grant_type %s not supported", conf.GrantType)
	}

	if conf.GrantType == "password" {
		if conf.Username == "" {
			return nil, fmt.Errorf("username must not be empty with grant_type=password")
		}
		if conf.Password == "" {
			return nil, fmt.Errorf("password must not be empty with grant_type=password")
		}
	} else {
		if conf.Username != "" {
			return nil, fmt.Errorf("username must not be set with grant_type=%s", conf.GrantType)
		}
		if conf.Password != "" {
			return nil, fmt.Errorf("password must not be set with grant_type=%s", conf.GrantType)
		}
	}

	assertionValue, err := eval.Value(evalCtx, conf.AssertionExpr)
	if err != nil {
		return nil, err
	}

	if conf.GrantType == "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		if assertionValue.IsNull() && assertionValue.Type() == cty.DynamicPseudoType {
			return nil, fmt.Errorf("missing assertion with grant_type=%s", conf.GrantType)
		}
	} else {
		if !(assertionValue.IsNull() && assertionValue.Type() == cty.DynamicPseudoType) {
			return nil, fmt.Errorf("assertion must not be set with grant_type=%s", conf.GrantType)
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

func (oa *OAuth2ReqAuth) GetToken(req *http.Request) error {
	requestContext := eval.ContextFromRequest(req).HCLContext()
	assertionValue, err := eval.Value(requestContext, oa.config.AssertionExpr)
	if err != nil {
		return err
	}

	formParams := url.Values{}

	if oa.config.GrantType == "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		if assertionValue.IsNull() {
			return fmt.Errorf("null assertion with grant_type=%s", oa.config.GrantType)
		} else if assertionValue.Type() != cty.String {
			return fmt.Errorf("assertion must evaluate to a string")
		} else {
			formParams.Set("assertion", assertionValue.AsString())
		}
	}

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

	if oa.config.Scope != "" {
		formParams.Set("scope", oa.config.Scope)
	}
	if oa.config.Password != "" || oa.config.Username != "" {
		formParams.Set("username", oa.config.Username)
		formParams.Set("password", oa.config.Password)
	}

	tokenResponseData, token, err := oa.oauth2Client.GetTokenResponse(req.Context(), formParams)
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
		err := oa.GetToken(req)
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

func (oa *OAuth2ReqAuth) value() (string, string) {
	token := oa.readAccessToken()
	return "oauth2", token
}
