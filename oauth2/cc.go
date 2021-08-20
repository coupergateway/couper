package oauth2

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

// CcClient represents an OAuth2 client using the client credentials flow.
type CcClient struct {
	*Client
}

// NewOAuth2CC creates a new OAuth2 Client Credentials client.
func NewOAuth2CC(conf *config.OAuth2ReqAuth, backend http.RoundTripper) (*CcClient, error) {
	backendErr := errors.Backend.Label(conf.Reference())
	if grantType := conf.GrantType; grantType != "client_credentials" {
		return nil, backendErr.Messagef("grant_type %s not supported", grantType)
	}

	if teAuthMethod := conf.TokenEndpointAuthMethod; teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, backendErr.Messagef("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	return &CcClient{&Client{backend, conf, conf}}, nil
}

// GetTokenResponse retrieves the response from the token endpoint
func (c *CcClient) GetTokenResponse(ctx context.Context) ([]byte, map[string]interface{}, string, error) {
	tokenResponse, tokenResponseData, accessToken, err := c.getTokenResponse(ctx, nil)
	if err != nil {
		return nil, nil, "", errors.Oauth2.Message("token request error").With(err)
	}

	return tokenResponse, tokenResponseData, accessToken, nil
}
