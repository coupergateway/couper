package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
)

// OAuth2 represents the transport <OAuth2> object.
type OAuth2 struct {
	Backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
}

// NewOAuth2CC() creates a new OAuth2 Client Credentials client.
func NewOAuth2CC(clientConf config.OAuth2Client, asConf config.OAuth2AS, backend http.RoundTripper) (*OAuth2, error) {
	backendErr := errors.Backend.Label(asConf.Reference())
	if grantType := clientConf.GetGrantType(); grantType != "client_credentials" {
		return nil, backendErr.Messagef("grant_type %s not supported", grantType)
	}

	if teAuthMethod := clientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, backendErr.Messagef("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	return &OAuth2{
		Backend:      backend,
		asConfig:     asConf,
		clientConfig: clientConf,
	}, nil
}

// NewOAuth2AC() creates a new OAuth2 Authorization Code client.
func NewOAuth2AC(clientConf config.OAuth2AcClient, asConf config.OAuth2AS, backend http.RoundTripper) (*OAuth2, error) {
	if grantType := clientConf.GetGrantType(); grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	if teAuthMethod := clientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	csrf := clientConf.GetCsrf()
	pkce := clientConf.GetPkce()
	if pkce == nil && csrf == nil {
		return nil, fmt.Errorf("CSRF protection not configured")
	}
	if csrf != nil {
		if csrf.TokenParam != "state" && csrf.TokenParam != "nonce" {
			return nil, fmt.Errorf("csrf_token_param %s not supported", csrf.TokenParam)
		}
		content, _, diags := csrf.HCLBody().PartialContent(csrf.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		csrf.Content = content
	}
	if pkce != nil {
		if pkce.CodeChallengeMethod != lib.CcmPlain && pkce.CodeChallengeMethod != lib.CcmS256 {
			return nil, fmt.Errorf("code_challenge_method %s not supported", pkce.CodeChallengeMethod)
		}
		content, _, diags := pkce.HCLBody().PartialContent(pkce.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		pkce.Content = content
	}
	return &OAuth2{
		Backend:      backend,
		asConfig:     asConf,
		clientConfig: clientConf,
	}, nil
}

func (oa *OAuth2) RequestToken(ctx context.Context, requestParams map[string]string) ([]byte, error) {
	tokenReq, err := oa.newTokenRequest(ctx, requestParams)
	if err != nil {
		return nil, err
	}

	tokenRes, err := oa.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, err
	}

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(oa.asConfig.Reference()).Message("token request read error").With(err)
	}

	if tokenRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(oa.asConfig.Reference()).Messagef("token request failed, response=%q", string(tokenResBytes))
	}

	return tokenResBytes, nil
}

func (oa *OAuth2) newTokenRequest(ctx context.Context, requestParams map[string]string) (*http.Request, error) {
	post := url.Values{}
	grantType := oa.clientConfig.GetGrantType()
	post.Set("grant_type", grantType)

	if scope := oa.clientConfig.GetScope(); scope != nil && grantType != "authorization_code" {
		post.Set("scope", *scope)
	}
	if requestParams != nil {
		for key, value := range requestParams {
			post.Set(key, value)
		}
	}
	teAuthMethod := oa.clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod != nil && *teAuthMethod == "client_secret_post" {
		post.Set("client_id", oa.clientConfig.GetClientID())
		post.Set("client_secret", oa.clientConfig.GetClientSecret())
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", nil)
	if err != nil {
		return nil, err
	}

	eval.SetBody(outreq, []byte(post.Encode()))

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if teAuthMethod == nil || *teAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(oa.clientConfig.GetClientID() + ":" + oa.clientConfig.GetClientSecret()))

		outreq.Header.Set("Authorization", "Basic "+auth)
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	if tokenURL := oa.asConfig.GetTokenEndpoint(); tokenURL != "" {
		outCtx = context.WithValue(outCtx, request.URLAttribute, tokenURL)
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
