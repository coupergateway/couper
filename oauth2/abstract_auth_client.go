package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

type AbstractAuthCodeClient struct {
	AuthCodeFlowClient
	*Client
	acClientConf config.OAuth2AcClient
}

// ExchangeCodeAndGetTokenResponse exchanges the authorization code and retrieves the response from the token endpoint.
func (a AbstractAuthCodeClient) ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	redirectURI := a.acClientConf.GetRedirectURI()
	if redirectURI == "" {
		return nil, errors.Oauth2.Message("redirect_uri is required")
	}

	absoluteURL, err := lib.AbsoluteURL(redirectURI, eval.NewRawOrigin(callbackURL))
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	formParams := url.Values{
		"code":         {code},
		"redirect_uri": {absoluteURL},
	}

	ctx := eval.ContextFromRequest(req).HCLContext()
	verifierVal, err := eval.ValueFromBodyAttribute(ctx, a.acClientConf.HCLBody(), "verifier_value")
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	verifierValue := strings.TrimSpace(seetie.ValueToString(verifierVal))
	if verifierValue == "" {
		return nil, errors.Oauth2.With(err).Message("Empty verifier_value")
	}

	verifierMethod, err := a.acClientConf.GetVerifierMethod()
	if err != nil {
		return nil, errors.Oauth2.With(err)
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
			return nil, errors.Oauth2.Messagef("missing state query parameter; query=%q", callbackURL.RawQuery)
		}

		if hashedVerifierValue != stateFromParam {
			return nil, errors.Oauth2.Messagef("state mismatch: %q (from query param) vs. %q (verifier_value: %q)", stateFromParam, hashedVerifierValue, verifierValue)
		}
	}

	tokenResponseData, accessToken, err := a.GetTokenResponse(req.Context(), formParams)
	if err != nil {
		return nil, errors.Oauth2.Message("token request error").With(err)
	}

	if err = a.validateTokenResponseData(req.Context(), tokenResponseData, hashedVerifierValue, verifierValue, accessToken); err != nil {
		return nil, errors.Oauth2.Message("token response validation error").With(err)
	}

	return tokenResponseData, nil
}

func Base64urlSha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
