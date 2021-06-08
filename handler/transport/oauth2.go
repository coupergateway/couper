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

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

// OAuth2 represents the transport <OAuth2> object.
type OAuth2 struct {
	Backend http.RoundTripper
	config  config.OAuth2
}

type OAuth2RequestConfig struct {
	Code         *string
	CodeVerifier *string
	CsrfToken    *string
	RedirectURI  *string
	StorageKey   string
}

// NewOAuth2 creates a new <OAuth2> object.
func NewOAuth2(conf config.OAuth2, backend http.RoundTripper) (*OAuth2, error) {
	if teAuthMethod := conf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("unsupported 'token_endpoint_auth_method': %s", *teAuthMethod)
		}
	}
	return &OAuth2{
		Backend: backend,
		config:  conf,
	}, nil
}

func (oa *OAuth2) GetRequestConfig(req *http.Request) (*OAuth2RequestConfig, error) {
	content, _, diags := oa.config.HCLBody().PartialContent(oa.config.Schema(true))
	if diags.HasErrors() {
		return nil, diags
	}

	evalContext, _ := req.Context().Value(eval.ContextType).(*eval.Context)

	var csrfToken, codeVerifier *string

	if v, ok := content.Attributes["code_verifier_value"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		codeVerifier = &strVal
	}

	if v, ok := content.Attributes["csrf_token_value"]; ok {
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
		csrfToken = &strVal
	}

	return &OAuth2RequestConfig{
		CodeVerifier: codeVerifier,
		CsrfToken:    csrfToken,
		// Backend is build up via config and token_endpoint will configure the backend,
		// use the backend memory location here.
		StorageKey: fmt.Sprintf("%p|%s|%s", &oa.Backend, oa.config.GetClientID(), oa.config.GetClientSecret()),
	}, nil
}

func (oa *OAuth2) RequestToken(ctx context.Context, requestConfig *OAuth2RequestConfig) ([]byte, error) {
	tokenReq, err := oa.newTokenRequest(ctx, requestConfig)
	if err != nil {
		return nil, err
	}

	tokenRes, err := oa.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, err
	}

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.Reference()).Message("token request read error").With(err)
	}

	if tokenRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(oa.config.Reference()).Messagef("token request failed, response='%s'", string(tokenResBytes))
	}

	return tokenResBytes, nil
}

func (oa *OAuth2) newTokenRequest(ctx context.Context, requestConfig *OAuth2RequestConfig) (*http.Request, error) {
	post := url.Values{}
	grantType := oa.config.GetGrantType()
	post.Set("grant_type", grantType)

	if scope := oa.config.GetScope(); scope != nil && grantType != "authorization_code" {
		post.Set("scope", *scope)
	}
	if requestConfig.RedirectURI != nil {
		post.Set("redirect_uri", *requestConfig.RedirectURI)
	}
	if requestConfig.Code != nil {
		post.Set("code", *requestConfig.Code)
	}
	if requestConfig.CodeVerifier != nil {
		post.Set("code_verifier", *requestConfig.CodeVerifier)
	}
	teAuthMethod := oa.config.GetTokenEndpointAuthMethod()
	if teAuthMethod != nil && *teAuthMethod == "client_secret_post" {
		post.Set("client_id", oa.config.GetClientID())
		post.Set("client_secret", oa.config.GetClientSecret())
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", nil)
	if err != nil {
		return nil, err
	}

	eval.SetBody(outreq, []byte(post.Encode()))

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if teAuthMethod == nil || *teAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(oa.config.GetClientID() + ":" + oa.config.GetClientSecret()))

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

func ParseAccessToken(jsonBytes []byte) (map[string]interface{}, string, error) {
	var jData map[string]interface{}

	err := json.Unmarshal(jsonBytes, &jData)
	if err != nil {
		return jData, "", err
	}

	var token string
	if t, ok := jData["access_token"].(string); ok {
		token = t
	} else {
		return jData, "", fmt.Errorf("missing access token")
	}

	return jData, token, nil
}
