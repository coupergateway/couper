package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/content"
	"github.com/avenga/couper/eval/lib"
)

// AcClient represents an OAuth2 client using the authorization code flow.
type AcClient interface {
	GetName() string
	GetTokenResponse(ctx context.Context, callbackURL *url.URL) (map[string]interface{}, error)
	validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error
}

type AbstractAcClient struct {
	AcClient
	Client
	name string
}

func (a AbstractAcClient) GetName() string {
	return a.name
}

// GetTokenResponse retrieves the response from the token endpoint
func (a AbstractAcClient) GetTokenResponse(ctx context.Context, callbackURL *url.URL) (map[string]interface{}, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	redirectURIVal, err := content.GetContextAttribute(a.clientConfig.HCLBody(), ctx, lib.RedirectURI)
	if redirectURIVal == "" {
		return nil, errors.Oauth2.Messagef("%s is required", lib.RedirectURI)
	}
	absoluteURL, err := lib.AbsoluteURL(redirectURIVal, eval.NewRawOrigin(callbackURL))
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	requestParams := map[string]string{
		"code":         code,
		"redirect_uri": absoluteURL,
	}

	verifierVal, err := content.GetContextAttribute(a.clientConfig.HCLBody(), ctx, "verifier_value")
	verifierValue := strings.TrimSpace(verifierVal)
	if verifierValue == "" {
		return nil, errors.Oauth2.Messagef("Empty verifier_value")
	}

	verifierMethod, err := getVerifierMethod(ctx, a.asConfig)
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	var hashedVerifierValue string
	if verifierMethod == config.CcmS256 {
		requestParams["code_verifier"] = verifierValue
	} else {
		hashedVerifierValue = Base64urlSha256(verifierValue)
	}

	if verifierMethod == "state" {
		stateFromParam := query.Get("state")
		if stateFromParam == "" {
			return nil, errors.Oauth2.Messagef("missing state query parameter; query=%q", callbackURL.RawQuery)
		}

		if hashedVerifierValue != stateFromParam {
			return nil, errors.Oauth2.Messagef("state mismatch: %q (from query param) vs. %q (verifier_value: %q)", stateFromParam, hashedVerifierValue, verifierValue)
		}
	}

	_, tokenResponseData, accessToken, err := a.getTokenResponse(ctx, requestParams)
	if err != nil {
		return nil, errors.Oauth2.Message("token request error").With(err)
	}

	if err = a.validateTokenResponseData(ctx, tokenResponseData, hashedVerifierValue, verifierValue, accessToken); err != nil {
		return nil, errors.Oauth2.Message("token response validation error").With(err)
	}

	return tokenResponseData, nil
}

// OAuth2AcClient represents an OAuth2 client using the (plain) authorization code flow.
type OAuth2AcClient struct {
	*AbstractAcClient
}

// NewOAuth2AC creates a new OAuth2 Authorization Code client.
func NewOAuth2AC(acClientConf config.OAuth2AcClient, oauth2AsConf config.OAuth2AS, backend http.RoundTripper) (*OAuth2AcClient, error) {
	if grantType := acClientConf.GetGrantType(); grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	if teAuthMethod := acClientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}

	switch acClientConf.(type) {
	case *config.OAuth2AC:
		verifierMethod, err := acClientConf.GetVerifierMethod("")
		if err != nil {
			return nil, err
		}

		if verifierMethod != config.CcmS256 && verifierMethod != "nonce" && verifierMethod != "state" {
			return nil, fmt.Errorf("verifier_method %s not supported", verifierMethod)
		}
	default:
		// skip this for oidc configurations due to possible startup errors
	}

	client := Client{
		Backend:      backend,
		asConfig:     oauth2AsConf,
		clientConfig: acClientConf,
	}
	o := &OAuth2AcClient{&AbstractAcClient{Client: client, name: acClientConf.GetName()}}
	o.AcClient = o
	return o, nil
}

// validateTokenResponseData validates the token response data (no-op)
func (o *OAuth2AcClient) validateTokenResponseData(_ context.Context, _ map[string]interface{}, _, _, _ string) error {
	return nil
}

func Base64urlSha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func getVerifierMethod(ctx context.Context, conf interface{}) (string, error) {
	clientConf, ok := conf.(config.OAuth2AcClient)
	if !ok {
		return "", fmt.Errorf("could not obtain verifier method configuration")
	}
	return clientConf.GetVerifierMethod(ctx.Value(request.UID).(string))
}
