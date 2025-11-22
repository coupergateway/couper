package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/hcl/v2"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
)

const (
	clientSecretBasic = "client_secret_basic"
	clientSecretJwt   = "client_secret_jwt"
	clientSecretPost  = "client_secret_post"
	privateKeyJwt     = "private_key_jwt"
)

// Client represents an OAuth2 client.
type Client struct {
	authenticator ClientAuthenticator
	backend       http.RoundTripper
	asConfig      config.OAuth2AS
	clientConfig  config.OAuth2Client
	grantType     string
}

func NewClient(evalCtx *hcl.EvalContext, grantType string, asConfig config.OAuth2AS, clientConfig config.OAuth2Client, backend http.RoundTripper) (*Client, error) {
	var authenticator ClientAuthenticator
	if clientConfig.ClientAuthenticationRequired() {
		tokenEndpoint, err := asConfig.GetTokenEndpoint()
		if err != nil {
			return nil, err
		}
		authenticator, err = NewClientAuthenticator(evalCtx, clientConfig.GetTokenEndpointAuthMethod(), "token_endpoint_auth_method", clientConfig.GetClientID(), clientConfig.GetClientSecret(), tokenEndpoint, clientConfig.GetJWTSigningProfile())
		if err != nil {
			return nil, err
		}
	}

	return &Client{
		authenticator,
		backend,
		asConfig,
		clientConfig,
		grantType,
	}, nil
}

func (c *Client) requestToken(tokenReq *http.Request) ([]byte, int, error) {
	ctx, cancel := context.WithCancel(tokenReq.Context())
	defer cancel()

	tokenRes, err := c.backend.RoundTrip(tokenReq.WithContext(ctx))
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

func (c *Client) newTokenRequest(ctx context.Context, formParams url.Values) (*http.Request, error) {
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

	formParams.Set("grant_type", c.grantType)

	if c.authenticator != nil {
		err = c.authenticator.Authenticate(&formParams, outreq)
		if err != nil {
			return nil, err
		}
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	eval.SetBody(outreq, []byte(formParams.Encode()))

	return outreq.WithContext(outCtx), nil
}

func (c *Client) GetTokenResponse(ctx context.Context, formParams url.Values) (map[string]interface{}, string, error) {
	tokenReq, err := c.newTokenRequest(ctx, formParams)
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
		errorURI, uExists := tokenResponseData["error_uri"].(string)
		if uExists {
			msg += fmt.Sprintf(", error_uri=%s", errorURI)
		}
		return nil, "", fmt.Errorf("%s", msg)
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
