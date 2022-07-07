package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/avenga/couper/config"
)

// AuthCodeFlowClient represents an OAuth2 client using the authorization code flow.
type AuthCodeFlowClient interface {
	GetName() string
	ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error)
	validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error
}

// AuthCodeClient represents an OAuth2 client using the (plain) authorization code flow.
type AuthCodeClient struct {
	*AbstractAuthCodeClient
}

// NewAuthCodeClient creates a new OAuth2 Authorization Code client.
func NewAuthCodeClient(acClientConf config.OAuth2AcClient, oauth2AsConf config.OAuth2AS, backend http.RoundTripper) (*AuthCodeClient, error) {
	grantType := acClientConf.GetGrantType()
	if grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	if teAuthMethod := acClientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}

	switch acClientConf.(type) {
	case *config.OAuth2AC:
		verifierMethod, err := acClientConf.GetVerifierMethod()
		if err != nil {
			return nil, err
		}

		if verifierMethod != config.CcmS256 && verifierMethod != "nonce" && verifierMethod != "state" {
			return nil, fmt.Errorf("verifier_method %s not supported", verifierMethod)
		}
	default:
		// skip this for oidc configurations due to possible startup errors
	}

	client := &Client{
		Backend:      backend,
		asConfig:     oauth2AsConf,
		clientConfig: acClientConf,
		grantType:    grantType,
	}
	o := &AuthCodeClient{&AbstractAuthCodeClient{
		Client: client,
		name:   acClientConf.GetName(),
	}}
	o.AuthCodeFlowClient = o
	return o, nil
}

// validateTokenResponseData validates the token response data (no-op)
func (o *AuthCodeClient) validateTokenResponseData(_ context.Context, _ map[string]interface{}, _, _, _ string) error {
	return nil
}
