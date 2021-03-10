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
	couperErr "github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
	"github.com/hashicorp/hcl/v2"
)

var _ http.RoundTripper = &OAuth2{}

// OAuth2 represents the transport <OAuth2> object.
type OAuth2 struct {
	backend  http.RoundTripper
	config   *config.OAuth2
	evalCtx  *hcl.EvalContext
	memStore *cache.MemoryStore
	next     http.RoundTripper
}

// NewOAuth2 creates a new <http.RoundTripper> object.
func NewOAuth2(
	evalCtx *hcl.EvalContext, config *config.OAuth2,
	memStore *cache.MemoryStore, backend, next http.RoundTripper,
) (http.RoundTripper, error) {
	if config.GrantType != "client_credentials" {
		return nil, fmt.Errorf("The grant_type has to be set to 'client_credentials'")
	} else if config.TokenEndpoint == "" {
		return nil, fmt.Errorf("Missing 'token_endpoint'")
	}

	_, err := url.Parse(config.TokenEndpoint)
	if err != nil {
		return nil, err
	}

	return &OAuth2{
		backend:  backend,
		config:   config,
		evalCtx:  evalCtx,
		memStore: memStore,
		next:     next,
	}, nil
}

// RoundTrip implements the <http.RoundTripper> interface.
func (oa *OAuth2) RoundTrip(req *http.Request) (*http.Response, error) {
	clientID, clientSecret, key, err := oa.getCredentials(req)
	if err != nil {
		return nil, err
	}

	if data := oa.memStore.Get(key); data != "" {
		token, err := oa.getAccessToken(data, key, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return oa.next.RoundTrip(req)
	}

	post := "grant_type=" + oa.config.GrantType
	body := ioutil.NopCloser(strings.NewReader(post))

	outreq, err := http.NewRequest("POST", oa.config.TokenEndpoint, body)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	outreq.Header.Set("Authorization", "Basic "+auth)
	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	outCtx := context.WithValue(outreq.Context(), request.TokenEndpoint, oa.config.TokenEndpoint)
	outCtx = context.WithValue(outCtx, request.UID, req.Context().Value(request.UID))
	*outreq = *outreq.WithContext(outCtx)

	outRes, err := oa.backend.RoundTrip(outreq)
	if err != nil {
		return nil, err
	}

	if outRes.StatusCode != http.StatusOK {
		return nil, couperErr.TokenRequestFailed
	}

	resBody, err := ioutil.ReadAll(outRes.Body)
	if err != nil {
		return nil, err
	}

	token, err := oa.getAccessToken(string(resBody), key, oa.memStore)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := oa.next.RoundTrip(req)

	if res.StatusCode == http.StatusUnauthorized {
		oa.memStore.Del(key)
	}

	return res, err
}

func (oa *OAuth2) getCredentials(req *http.Request) (string, string, string, error) {
	content, _, diags := oa.config.Remain.PartialContent(
		oa.config.Schema(true),
	)
	if diags.HasErrors() {
		return "", "", "", diags
	}

	evalCtx := eval.NewHTTPContext(oa.evalCtx, eval.BufferNone, req)

	id, idOK := content.Attributes["client_id"]
	idv, _ := id.Expr.Value(evalCtx)
	clientID := seetie.ValueToString(idv)

	secret, secretOK := content.Attributes["client_secret"]
	secretv, _ := secret.Expr.Value(evalCtx)
	clientSecret := seetie.ValueToString(secretv)

	if !idOK || !secretOK {
		return "", "", "", couperErr.MissingOAuth2Credentials
	}

	key := oa.config.TokenEndpoint + "|" + clientID + "|" + clientSecret

	return clientID, clientSecret, key, nil
}

func (oa *OAuth2) getAccessToken(jsonString, key string, memStore *cache.MemoryStore) (string, error) {
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

	if memStore != nil {
		var ttl int64
		if t, ok := jData["expires_in"].(float64); ok {
			ttl = (int64)(t * 0.9)
		}

		memStore.Set(key, jsonString, ttl)
	}

	return token, nil
}
