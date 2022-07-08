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
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

type AbstractAuthCodeClient struct {
	AuthCodeFlowClient
	*Client
	name string
}

func (a AbstractAuthCodeClient) GetName() string {
	return a.name
}

// ExchangeCodeAndGetTokenResponse exchanges the authorization code and retrieves the response from the token endpoint.
func (a AbstractAuthCodeClient) ExchangeCodeAndGetTokenResponse(req *http.Request, callbackURL *url.URL) (map[string]interface{}, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	ctx := eval.ContextFromRequest(req).HCLContext()

	redirectURIVal, err := eval.ValueFromBodyAttribute(ctx, a.clientConfig.HCLBody(), lib.RedirectURI)
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	redirectURI := seetie.ValueToString(redirectURIVal)
	if redirectURI == "" {
		return nil, errors.Oauth2.With(err).Messagef("%s is required", lib.RedirectURI)
	}

	absoluteURL, err := lib.AbsoluteURL(redirectURI, eval.NewRawOrigin(callbackURL))
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	formParams := url.Values{
		"code":         {code},
		"redirect_uri": {absoluteURL},
	}

	verifierVal, err := eval.ValueFromBodyAttribute(ctx, a.clientConfig.HCLBody(), "verifier_value")
	if err != nil {
		return nil, errors.Oauth2.With(err)
	}

	verifierValue := strings.TrimSpace(seetie.ValueToString(verifierVal))
	if verifierValue == "" {
		return nil, errors.Oauth2.With(err).Messagef("Empty verifier_value")
	}

	verifierMethod, err := getVerifierMethod(req.Context(), a.asConfig)
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

func getVerifierMethod(ctx context.Context, conf interface{}) (string, error) {
	clientConf, ok := conf.(config.OAuth2AcClient)
	if !ok {
		return "", fmt.Errorf("could not obtain verifier method configuration")
	}
	return clientConf.GetVerifierMethod()
}
