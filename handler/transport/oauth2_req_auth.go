package transport

import (
	"context"
	"fmt"
	"net/http"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/oauth2"
)

var _ http.RoundTripper = &OAuth2ReqAuth{}

// OAuth2ReqAuth represents the transport <OAuth2ReqAuth> object.
type OAuth2ReqAuth struct {
	oauth2Client *oauth2.Client
	config       *config.OAuth2ReqAuth
	memStore     *cache.MemoryStore
	next         http.RoundTripper
}

// NewOAuth2ReqAuth creates a new <http.RoundTripper> object.
func NewOAuth2ReqAuth(conf *config.OAuth2ReqAuth, memStore *cache.MemoryStore,
	oauth2Client *oauth2.Client, next http.RoundTripper) (http.RoundTripper, error) {
	return &OAuth2ReqAuth{
		config:       conf,
		oauth2Client: oauth2Client,
		memStore:     memStore,
		next:         next,
	}, nil
}

// RoundTrip implements the <http.RoundTripper> interface.
func (oa *OAuth2ReqAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	storageKey := fmt.Sprintf("%p|%s|%s", &oa.oauth2Client.Backend, oa.config.ClientID, oa.config.ClientSecret)
	if data := oa.memStore.Get(storageKey); data != "" {
		token, terr := oa.readAccessToken(data)
		if terr != nil {
			return nil, errors.Backend.Label(oa.config.BackendName).Message("token read error").With(terr)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return oa.next.RoundTrip(req)
	}

	ctx := req.Context()
	tokenResponse, tokenResponseData, token, err := oa.oauth2Client.GetTokenResponse(ctx)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).Message("token request error").With(err)
	}

	oa.updateAccessToken(tokenResponse, tokenResponseData, storageKey)

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := oa.next.RoundTrip(req)

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		oa.memStore.Del(storageKey)

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
	_, token, err := oauth2.ParseTokenResponse([]byte(data))
	if err != nil {
		return "", err
	}

	return token, nil
}

func (oa *OAuth2ReqAuth) updateAccessToken(jsonBytes []byte, jData map[string]interface{}, key string) {
	if oa.memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		oa.memStore.Set(key, string(jsonBytes), ttl)
	}
}
