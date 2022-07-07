package oauth2

import (
	"context"
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
	grantType    string
}

func (c *Client) requestToken(tokenReq *http.Request) ([]byte, int, error) {
	tokenRes, err := c.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, 0, err
	}
	defer tokenRes.Body.Close()

	tokenResBytes, err := io.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, tokenRes.StatusCode, err
	}

	return tokenResBytes, tokenRes.StatusCode, nil
}

func (c *Client) newTokenRequest(ctx context.Context, requestParams map[string]string) (*http.Request, error) {
	post := url.Values{}
	post.Set("grant_type", c.grantType)

	if scope := c.clientConfig.GetScope(); scope != "" && c.grantType != "authorization_code" {
		post.Set("scope", scope)
	}

	for key, value := range requestParams {
		post.Set(key, value)
	}

	teAuthMethod := c.clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod != nil && *teAuthMethod == "client_secret_post" {
		post.Set("client_id", c.clientConfig.GetClientID())
		post.Set("client_secret", c.clientConfig.GetClientSecret())
	}

	tokenURL, err := c.asConfig.GetTokenEndpoint()
	if err != nil {
		return nil, err
	}

	outreq, err := http.NewRequest(http.MethodPost, tokenURL, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Accept", "application/json")
	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if teAuthMethod == nil || *teAuthMethod == "client_secret_basic" {
		outreq.SetBasicAuth(url.QueryEscape(c.clientConfig.GetClientID()), url.QueryEscape(c.clientConfig.GetClientSecret()))
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	eval.SetBody(outreq, []byte(post.Encode()))

	return outreq.WithContext(outCtx), nil
}

func (c *Client) GetTokenResponse(ctx context.Context, requestParams map[string]string) (map[string]interface{}, string, error) {
	tokenReq, err := c.newTokenRequest(ctx, requestParams)
	if err != nil {
		return nil, "", err
	}

	tokenResponse, statusCode, err := c.requestToken(tokenReq)
	if err != nil {
		return nil, "", err
	}

	tokenResponseData, accessToken, err := parseTokenResponse(tokenResponse)
	if err != nil {
		return nil, "", err
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
		return nil, "", fmt.Errorf(msg)
	}

	return tokenResponseData, accessToken, nil
}

func parseTokenResponse(tokenResponse []byte) (map[string]interface{}, string, error) {
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
