package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
)

// AuthCodeFlowClient represents an OAuth2 client using the authorization code flow.
type AuthCodeFlowClient interface {
	ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error)
	validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error
}

// AuthCodeClient represents an OAuth2 client using the (plain) authorization code flow.
type AuthCodeClient struct {
	*AbstractAuthCodeClient
}

// NewAuthCodeClient creates a new OAuth2 Authorization Code client.
func NewAuthCodeClient(evalCtx *hcl.EvalContext, acClientConf config.OAuth2AcClient, oauth2AsConf config.OAuth2AS, backend http.RoundTripper) (*AuthCodeClient, error) {
	grantType := acClientConf.GetGrantType()
	if grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	switch acClientConf.(type) {
	case *config.OAuth2AC:
		verifierMethod, err := acClientConf.GetVerifierMethod()
		if err != nil {
			return nil, err
		}

		if verifierMethod != config.CcmS256 && verifierMethod != "state" {
			return nil, fmt.Errorf("verifier_method %s not supported", verifierMethod)
		}
	default:
		// skip this for oidc configurations due to possible startup errors
	}

	client, err := NewClient(evalCtx, grantType, oauth2AsConf, acClientConf, backend)
	if err != nil {
		return nil, err
	}

	o := &AuthCodeClient{&AbstractAuthCodeClient{
		acClientConf: acClientConf,
		Client:       client,
	}}
	o.AuthCodeFlowClient = o
	return o, nil
}

// validateTokenResponseData validates the token response data (no-op)
func (o *AuthCodeClient) validateTokenResponseData(_ context.Context, _ map[string]interface{}, _, _, _ string) error {
	return nil
}
