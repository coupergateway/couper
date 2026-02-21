package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/seetie"
)

// AuthCodeFlowClient represents an OAuth2 client using the authorization code flow.
type AuthCodeFlowClient interface {
	// ExchangeCodeAndGetTokenResponse exchanges the authorization code and retrieves the response from the token endpoint.
	ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error)
}

var (
	_ AuthCodeFlowClient = &AuthCodeClient{}
)

// AuthCodeClient represents an OAuth2 client using the (plain) authorization code flow.
type AuthCodeClient struct {
	*Client
	acClientConf config.OAuth2AcClient
}

// NewAuthCodeClient creates a new OAuth2 Authorization Code client.
func NewAuthCodeClient(evalCtx *hcl.EvalContext, acClientConf config.OAuth2AcClient, oauth2AsConf config.OAuth2AS, backend http.RoundTripper, name string) (*AuthCodeClient, error) {
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

	client, err := NewClient(evalCtx, grantType, oauth2AsConf, acClientConf, backend, name)
	if err != nil {
		return nil, err
	}

	o := &AuthCodeClient{
		acClientConf: acClientConf,
		Client:       client,
	}
	return o, nil
}

// ExchangeCodeAndGetTokenResponse exchanges the authorization code and retrieves the response from the token endpoint.
func (a *AuthCodeClient) ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error) {
	tokenResponseData, _, _, _, err := a.exchangeCodeAndGetTokenResponse(req, callbackURL)
	if err != nil {
		return nil, err
	}

	return tokenResponseData, nil
}

func (a *AuthCodeClient) exchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, string, string, string, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, "", "", "", errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	redirectURI := a.acClientConf.GetRedirectURI()
	if redirectURI == "" {
		return nil, "", "", "", errors.Oauth2.Message("redirect_uri is required")
	}

	absoluteURL, err := lib.AbsoluteURL(redirectURI, eval.NewRawOrigin(callbackURL))
	if err != nil {
		return nil, "", "", "", errors.Oauth2.With(err)
	}

	formParams := url.Values{
		"code":         {code},
		"redirect_uri": {absoluteURL},
	}

	ctx := eval.ContextFromRequest(req).HCLContext()
	verifierVal, err := eval.ValueFromBodyAttribute(ctx, a.acClientConf.HCLBody(), "verifier_value")
	if err != nil {
		return nil, "", "", "", errors.Oauth2.With(err)
	}

	verifierValue := strings.TrimSpace(seetie.ValueToString(verifierVal))
	if verifierValue == "" {
		return nil, "", "", "", errors.Oauth2.With(err).Message("Empty verifier_value")
	}

	verifierMethod, err := a.acClientConf.GetVerifierMethod()
	if err != nil {
		return nil, "", "", "", errors.Oauth2.With(err)
	}

	var hashedVerifierValue string
	if verifierMethod == config.CcmS256 {
		formParams.Set("code_verifier", verifierValue)
	} else {
		hashedVerifierValue = Base64urlSha256(verifierValue)
	}

	if verifierMethod == "state" {
		stateFromParam := query.Get("state")
		if stateFromParam == "" {
			return nil, "", "", "", errors.Oauth2.Messagef("missing state query parameter; query=%q", callbackURL.RawQuery)
		}

		if hashedVerifierValue != stateFromParam {
			return nil, "", "", "", errors.Oauth2.Messagef("state mismatch: %q (from query param) vs. %q (verifier_value: %q)", stateFromParam, hashedVerifierValue, verifierValue)
		}
	}

	tokenResponseData, accessToken, err := a.GetTokenResponse(req.Context(), formParams)
	if err != nil {
		return nil, "", "", "", errors.Oauth2.Message("token request error").With(err)
	}

	return tokenResponseData, hashedVerifierValue, verifierValue, accessToken, nil
}

// Base64urlSha256 creates a base64url encoded sha256 hash of the given input string.
func Base64urlSha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
