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

var supportedGrantTypes = map[string]struct{}{
	config.ClientCredentials: {},
	config.JwtBearer:         {},
	config.Password:          {},
}

var (
	_ RequestAuthorizer = &OAuth2ReqAuth{}
)

// OAuth2ReqAuth represents the transport <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	config       *config.OAuth2ReqAuth
	mu           sync.Mutex
	memStore     *cache.MemoryStore
	oauth2Client *oauth2.Client
	storageKey   string
}

// NewOAuth2ReqAuth implements the http.RoundTripper interface to wrap an existing Backend / http.RoundTripper
// to retrieve a valid token before passing the initial out request.
func NewOAuth2ReqAuth(evalCtx *hcl.EvalContext, conf *config.OAuth2ReqAuth, memStore *cache.MemoryStore,
	asBackend http.RoundTripper) (RequestAuthorizer, error) {

	if _, supported := supportedGrantTypes[conf.GrantType]; !supported {
		return nil, fmt.Errorf("grant_type %s not supported", conf.GrantType)
	}

	if conf.GrantType == config.Password {
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

	if conf.GrantType == config.JwtBearer {
		if assertionValue.IsNull() && assertionValue.Type() == cty.DynamicPseudoType {
			return nil, fmt.Errorf("missing assertion with grant_type=%s", conf.GrantType)
		}
	} else {
		if !(assertionValue.IsNull() && assertionValue.Type() == cty.DynamicPseudoType) {
			return nil, fmt.Errorf("assertion must not be set with grant_type=%s", conf.GrantType)
		}
	}

	oauth2Client, err := oauth2.NewClient(evalCtx, conf.GrantType, conf, conf, asBackend)
	if err != nil {
		return nil, err
	}

	reqAuth := &OAuth2ReqAuth{
		config:       conf,
		oauth2Client: oauth2Client,
		memStore:     memStore,
	}
	reqAuth.storageKey = fmt.Sprintf("oauth2-%p", reqAuth)
	return reqAuth, nil
}

func (oa *OAuth2ReqAuth) GetToken(req *http.Request) error {
	token := oa.readAccessToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	oa.mu.Lock()
	defer oa.mu.Unlock()

	token = oa.readAccessToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	requestError := errors.Request.Label("oauth2")
	formParams := url.Values{}

	if oa.config.GrantType == config.JwtBearer {
		requestContext := eval.ContextFromRequest(req).HCLContext()
		assertionValue, err := eval.Value(requestContext, oa.config.AssertionExpr)
		if err != nil {
			return err
		}

		if assertionValue.IsNull() {
			return requestError.Message("assertion expression evaluates to null")
		} else if assertionValue.Type() != cty.String {
			return requestError.Message("assertion expression must evaluate to a string")
		} else {
			formParams.Set("assertion", assertionValue.AsString())
		}
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
		return requestError.Message("token request failed") // don't propagate token request roundtrip error
	}

	oa.updateAccessToken(token, tokenResponseData)

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
