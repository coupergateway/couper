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
	"github.com/avenga/couper/eval"
)

// Client represents an OAuth2 client.
type Client struct {
	Backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
}

func (c *Client) requestToken(ctx context.Context, requestParams map[string]string) ([]byte, int, error) {
	tokenReq, err := c.newTokenRequest(ctx, requestParams)
	if err != nil {
		return nil, 0, err
	}

	tokenRes, err := c.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, 0, err
	}

	tokenResBytes, err := io.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, 0, err
	}

	return tokenResBytes, tokenRes.StatusCode, nil
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
	tokenResponse, statusCode, err := c.requestToken(ctx, requestParams)
	if err != nil {
		return nil, nil, "", err
	}

	tokenResponseData, accessToken, err := ParseTokenResponse(tokenResponse)
	if err != nil {
		return nil, nil, "", err
	}

	if statusCode != http.StatusOK {
		e, _ := tokenResponseData["error"].(string)
		msg := fmt.Sprintf("error=%s", e)
		errorDescription, dExists := tokenResponseData["error_description"].(string)
		if dExists {
			msg += fmt.Sprintf(", error_description=%s", errorDescription)
		}
		errorUri, uExists := tokenResponseData["error_uri"].(string)
		if uExists {
			msg += fmt.Sprintf(", error_uri=%s", errorUri)
		}
		return nil, nil, "", fmt.Errorf(msg)
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
		accessToken = ""
	}

	return tokenResponseData, accessToken, nil
}
