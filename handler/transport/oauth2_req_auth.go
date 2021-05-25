package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

var _ http.RoundTripper = &OAuth2ReqAuth{}

// OAuth2ReqAuth represents the transport <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	oauth2   *OAuth2
	config   *config.OAuth2
	memStore *cache.MemoryStore
	next     http.RoundTripper
}

// NewOAuth2 creates a new <http.RoundTripper> object.
func NewOAuth2ReqAuth(conf *config.OAuth2, memStore *cache.MemoryStore,
	oauth2 *OAuth2, next http.RoundTripper) (http.RoundTripper, error) {
	const grantType = "client_credentials"
	if conf.GrantType != grantType {
		return nil, errors.Backend.Label(conf.BackendName).Message("grant_type not supported: " + conf.GrantType)
	}

	return &OAuth2ReqAuth{
		config:   conf,
		oauth2:   oauth2,
		memStore: memStore,
		next:     next,
	}, nil
}

// RoundTrip implements the <http.RoundTripper> interface.
func (oa *OAuth2ReqAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	requestConfig, err := oa.oauth2.GetRequestConfig(req)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).With(err)
	}

	if data := oa.memStore.Get(requestConfig.StorageKey); data != "" {
		token, terr := oa.readAccessToken(data)
		if terr != nil {
			return nil, errors.Backend.Label(oa.config.BackendName).Message("token read error").With(terr)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return oa.next.RoundTrip(req)
	}

	tokenResponse, err := oa.oauth2.RequestToken(req.Context(), requestConfig)

	token, err := oa.updateAccessToken(tokenResponse, requestConfig.StorageKey)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).Message("token update error").With(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := oa.next.RoundTrip(req)

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		oa.memStore.Del(requestConfig.StorageKey)

		ctx := req.Context()
		if retries, ok := ctx.Value(request.TokenRequestRetries).(uint8); !ok || retries < *oa.config.Retries {
			ctx = context.WithValue(ctx, request.TokenRequestRetries, retries+1)

			req.Header.Del("Authorization")
			*req = *req.WithContext(ctx)

			return oa.RoundTrip(req)
		}
	}

	return res, err
}

func (oa *OAuth2ReqAuth) readAccessToken(data string) (string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal([]byte(data), &jData)
	if err != nil {
		return "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return "", fmt.Errorf("missing access token")
	}

	return token, nil
}

func (oa *OAuth2ReqAuth) updateAccessToken(jsonString, key string) (string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal([]byte(jsonString), &jData)
	if err != nil {
		return "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return "", fmt.Errorf("missing access token")
	}

	if oa.memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		oa.memStore.Set(key, jsonString, ttl)
	}

	return token, nil
}
