package oauth2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
)

// Client represents an OAuth2 client.
type Client struct {
	Backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
}

func (c *Client) requestToken(ctx context.Context, requestParams map[string]string) ([]byte, error) {
	tokenReq, err := c.newTokenRequest(ctx, requestParams)
	if err != nil {
		return nil, err
	}

	tokenRes, err := c.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, err
	}

	tokenResBytes, err := io.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(c.asConfig.Reference()).Message("token request read error").With(err)
	}

	if tokenRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(c.asConfig.Reference()).Messagef("token request failed, response=%q", string(tokenResBytes))
	}

	return tokenResBytes, nil
}

func (c *Client) newTokenRequest(ctx context.Context, requestParams map[string]string) (*http.Request, error) {
	post := url.Values{}
	grantType := c.clientConfig.GetGrantType()
	post.Set("grant_type", grantType)

	if scope := c.clientConfig.GetScope(); scope != "" && grantType != "authorization_code" {
		post.Set("scope", scope)
	}
	if requestParams != nil {
		for key, value := range requestParams {
			post.Set(key, value)
		}
	}
	teAuthMethod := c.clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod != nil && *teAuthMethod == "client_secret_post" {
		post.Set("client_id", c.clientConfig.GetClientID())
		post.Set("client_secret", c.clientConfig.GetClientSecret())
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", nil)
	if err != nil {
		return nil, err
	}

	eval.SetBody(outreq, []byte(post.Encode()))

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if teAuthMethod == nil || *teAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.clientConfig.GetClientID() + ":" + c.clientConfig.GetClientSecret()))

		outreq.Header.Set("Authorization", "Basic "+auth)
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	tokenURL, err := c.asConfig.GetTokenEndpoint()
	if err != nil {
		return nil, err
	}

	if tokenURL != "" {
		outCtx = context.WithValue(outCtx, request.URLAttribute, tokenURL)
	}

	return outreq.WithContext(outCtx), nil
}

func (c *Client) getTokenResponse(ctx context.Context, requestParams map[string]string) ([]byte, map[string]interface{}, string, error) {
	tokenResponse, err := c.requestToken(ctx, requestParams)
	if err != nil {
		return nil, nil, "", err
	}

	tokenResponseData, accessToken, err := ParseTokenResponse(tokenResponse)
	if err != nil {
		return nil, nil, "", errors.Oauth2.Messagef("parsing token response JSON failed, response=%q", string(tokenResponse)).With(err)
	}

	return tokenResponse, tokenResponseData, accessToken, nil
}

func ParseTokenResponse(tokenResponse []byte) (map[string]interface{}, string, error) {
	var tokenResponseData map[string]interface{}

	err := json.Unmarshal(tokenResponse, &tokenResponseData)
	if err != nil {
		return nil, "", err
	}

	var accessToken string
	if t, ok := tokenResponseData["access_token"].(string); ok {
		accessToken = t
	} else {
		return nil, "", fmt.Errorf("missing access token")
	}

	return tokenResponseData, accessToken, nil
}
