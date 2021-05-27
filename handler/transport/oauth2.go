package transport

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

// OAuth2 represents the transport <OAuth2> object.
type OAuth2 struct {
	backend http.RoundTripper
	config  config.OAuth2
}

type OAuth2RequestConfig struct {
	ClientID                string
	ClientSecret            string
	Code                    string
	CodeVerifier            *string
	RedirectURI             *string
	Scope                   *string
	StorageKey              string
	TokenEndpointAuthMethod *string
}

// NewOAuth2 creates a new <OAuth2> object.
func NewOAuth2(conf config.OAuth2, backend http.RoundTripper) (*OAuth2, error) {
	return &OAuth2{
		backend: backend,
		config:  conf,
	}, nil
}

func (oa *OAuth2) GetRequestConfig(req *http.Request) (*OAuth2RequestConfig, error) {
	content, _, diags := oa.config.HCLBody().PartialContent(oa.config.Schema(true))
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

	var codeVerifier, redirectUri, scope, teAuthMethod *string

	if v, ok := content.Attributes["code_verifier_value"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		codeVerifier = &strVal
	}

	if v, ok := content.Attributes["redirect_uri"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		redirectUri = &strVal
	}

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

	return &OAuth2RequestConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectUri,
		Scope:        scope,
		// Backend is build up via config and token_endpoint will configure the backend,
		// use the backend memory location here.
		StorageKey:              fmt.Sprintf("%p|%s|%s", &oa.backend, clientID, clientSecret),
		TokenEndpointAuthMethod: teAuthMethod,
	}, nil
}

func (oa *OAuth2) RequestToken(ctx context.Context, requestConfig *OAuth2RequestConfig) (string, error) {
	tokenReq, err := oa.newTokenRequest(ctx, requestConfig)
	if err != nil {
		return "", err
	}

	tokenRes, err := oa.backend.RoundTrip(tokenReq)
	if err != nil {
		return "", err
	}

	if tokenRes.StatusCode != http.StatusOK {
		return "", errors.Backend.Label(oa.config.Reference()).Message("token request failed")
	}

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		return "", errors.Backend.Label(oa.config.Reference()).Message("token request read error").With(err)
	}

	return string(tokenResBytes), nil
}

func (oa *OAuth2) newTokenRequest(ctx context.Context, requestConfig *OAuth2RequestConfig) (*http.Request, error) {
	post := url.Values{}
	post.Set("grant_type", oa.config.GetGrantType())

	if requestConfig.Scope != nil {
		post.Set("scope", *requestConfig.Scope)
	}
	if requestConfig.RedirectURI != nil {
		post.Set("redirect_uri", *requestConfig.RedirectURI)
	}
	if requestConfig.CodeVerifier != nil {
		post.Set("code_verifier", *requestConfig.CodeVerifier)
	}
	if requestConfig.TokenEndpointAuthMethod != nil && *requestConfig.TokenEndpointAuthMethod == "client_secret_post" {
		post.Set("client_id", requestConfig.ClientID)
		post.Set("client_secret", requestConfig.ClientSecret)
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", nil)
	if err != nil {
		return nil, err
	}

	eval.SetBody(outreq, []byte(post.Encode()))

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if requestConfig.TokenEndpointAuthMethod == nil || *requestConfig.TokenEndpointAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(requestConfig.ClientID + ":" + requestConfig.ClientSecret))

		outreq.Header.Set("Authorization", "Basic "+auth)
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	url, err := eval.GetContextAttribute(oa.config.HCLBody(), outCtx, "token_endpoint")
	if err != nil {
		return nil, err
	}

	if url != "" {
		outCtx = context.WithValue(outCtx, request.URLAttribute, url)
	}

	return outreq.WithContext(outCtx), nil
}
