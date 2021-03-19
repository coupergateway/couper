package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	couperErr "github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

var _ http.RoundTripper = &OAuth2{}

// OAuth2 represents the transport <OAuth2> object.
type OAuth2 struct {
	backend  http.RoundTripper
	config   *config.OAuth2
	memStore *cache.MemoryStore
	next     http.RoundTripper
}

type OAuth2Credentials struct {
	ClientID     string
	ClientSecret string
	StorageKey   string
}

// NewOAuth2 creates a new <http.RoundTripper> object.
func NewOAuth2(config *config.OAuth2, memStore *cache.MemoryStore,
	backend, next http.RoundTripper) (http.RoundTripper, error) {
	if config.GrantType != "client_credentials" {
		return nil, fmt.Errorf("the grant_type has to be set to 'client_credentials'")
	}

	return &OAuth2{
		backend:  backend,
		config:   config,
		memStore: memStore,
		next:     next,
	}, nil
}

// RoundTrip implements the <http.RoundTripper> interface.
func (oa *OAuth2) RoundTrip(req *http.Request) (*http.Response, error) {
	credentials, err := oa.getCredentials(req)
	if err != nil {
		return nil, err
	}

	if data := oa.memStore.Get(credentials.StorageKey); data != "" {
		token, err := oa.readAccessToken(data)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return oa.next.RoundTrip(req)
	}

	tokenReq, err := oa.newTokenRequest(req.Context(), credentials)
	if err != nil {
		return nil, err
	}

	tokenRes, err := oa.backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, err
	}

	if tokenRes.StatusCode != http.StatusOK {
		return nil, couperErr.TokenRequestFailed
	}

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, err
	}

	token, err := oa.updateAccessToken(string(tokenResBytes), credentials.StorageKey)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := oa.next.RoundTrip(req)

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		oa.memStore.Del(credentials.StorageKey)
	}

	return res, err
}

func (oa *OAuth2) getCredentials(req *http.Request) (*OAuth2Credentials, error) {
	content, _, diags := oa.config.Remain.PartialContent(oa.config.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	evalContext, _ := req.Context().Value(eval.ContextType).(*eval.Context)

	id, idOK := content.Attributes["client_id"]
	idv, _ := id.Expr.Value(evalContext.HCLContext())
	clientID := seetie.ValueToString(idv)

	secret, secretOK := content.Attributes["client_secret"]
	secretv, _ := secret.Expr.Value(evalContext.HCLContext())
	clientSecret := seetie.ValueToString(secretv)

	if !idOK || !secretOK {
		return nil, couperErr.MissingOAuth2Credentials
	}

	return &OAuth2Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		// Backend is build up via config and token_endpoint will configure the backend,
		// use the backend memory location here.
		StorageKey: fmt.Sprintf("%p|%s|%s", &oa.backend, clientID, clientSecret),
	}, nil
}

func (oa *OAuth2) newTokenRequest(ctx context.Context, creds *OAuth2Credentials) (*http.Request, error) {
	post := "grant_type=" + oa.config.GrantType
	body := ioutil.NopCloser(strings.NewReader(post))

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest("POST", "", body)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(creds.ClientID + ":" + creds.ClientSecret))
	outreq.Header.Set("Authorization", "Basic "+auth)
	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	url, err := eval.GetContextAttribute(oa.config.Remain, outCtx, "token_endpoint")
	if err != nil {
		return nil, err
	}

	if url != "" {
		outCtx = context.WithValue(outCtx, request.URLAttribute, url)
	}

	return outreq.WithContext(outCtx), nil
}

func (oa *OAuth2) readAccessToken(data string) (string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal([]byte(data), &jData)
	if err != nil {
		return "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return "", couperErr.MissingOAuth2AccessToken
	}

	return token, nil
}

func (oa *OAuth2) updateAccessToken(jsonString, key string) (string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal([]byte(jsonString), &jData)
	if err != nil {
		return "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return "", couperErr.MissingOAuth2AccessToken
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
