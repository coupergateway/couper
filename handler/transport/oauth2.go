package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
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
	ClientID                string
	ClientSecret            string
	Scope                   *string
	StorageKey              string
	TokenEndpointAuthMethod *string
}

// NewOAuth2 creates a new <http.RoundTripper> object.
func NewOAuth2(conf *config.OAuth2, memStore *cache.MemoryStore,
	backend, next http.RoundTripper) (http.RoundTripper, error) {
	const grantType = "client_credentials"
	if conf.GrantType != grantType {
		return nil, errors.Backend.Label(conf.BackendName).Message("grant_type not supported: " + conf.GrantType)
	}

	return &OAuth2{
		backend:  backend,
		config:   conf,
		memStore: memStore,
		next:     next,
	}, nil
}

// RoundTrip implements the <http.RoundTripper> interface.
func (oa *OAuth2) RoundTrip(req *http.Request) (*http.Response, error) {
	credentials, err := oa.getCredentials(req)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).With(err)
	}

	if data := oa.memStore.Get(credentials.StorageKey); data != "" {
		token, terr := oa.readAccessToken(data)
		if terr != nil {
			return nil, errors.Backend.Label(oa.config.BackendName).Message("token read error").With(terr)
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
		return nil, errors.Backend.Label(oa.config.BackendName).Message("token request failed")
	}

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).Message("token request read error").With(err)
	}

	token, err := oa.updateAccessToken(string(tokenResBytes), credentials.StorageKey)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.BackendName).Message("token update error").With(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := oa.next.RoundTrip(req)

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		oa.memStore.Del(credentials.StorageKey)

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
		return nil, fmt.Errorf("missing credentials")
	}

	var scope, teAuthMethod *string

	if v, ok := content.Attributes["scope"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		scope = &strVal
	}

	if v, ok := content.Attributes["token_endpoint_auth_method"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		teAuthMethod = &strVal
	}

	if teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("unsupported 'token_endpoint_auth_method': %s", *teAuthMethod)
		}
	}

	return &OAuth2Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scope:        scope,
		// Backend is build up via config and token_endpoint will configure the backend,
		// use the backend memory location here.
		StorageKey:              fmt.Sprintf("%p|%s|%s", &oa.backend, clientID, clientSecret),
		TokenEndpointAuthMethod: teAuthMethod,
	}, nil
}

func (oa *OAuth2) newTokenRequest(ctx context.Context, creds *OAuth2Credentials) (*http.Request, error) {
	post := url.Values{}
	post.Set("grant_type", oa.config.GrantType)

	if creds.Scope != nil {
		post.Set("scope", *creds.Scope)
	}
	if creds.TokenEndpointAuthMethod != nil && *creds.TokenEndpointAuthMethod == "client_secret_post" {
		post.Set("client_id", creds.ClientID)
		post.Set("client_secret", creds.ClientSecret)
	}

	body := ioutil.NopCloser(strings.NewReader(post.Encode()))

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", body)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if creds.TokenEndpointAuthMethod == nil || *creds.TokenEndpointAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(creds.ClientID + ":" + creds.ClientSecret))

		outreq.Header.Set("Authorization", "Basic "+auth)
	}

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
		return "", fmt.Errorf("missing access token")
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
