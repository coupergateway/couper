package lib

import (
	"fmt"
	"net/url"

	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/avenga/couper/config"
)

const (
	FnOAuthAuthorizationUrl = "beta_oauth_authorization_url"
	FnOAuthCodeVerifier     = "beta_oauth_code_verifier"
	FnOAuthCodeChallenge    = "beta_oauth_code_challenge"
	FnOAuthCsrfToken        = "beta_oauth_csrf_token"
	FnOAuthHashedCsrfToken  = "beta_oauth_hashed_csrf_token"
	CodeVerifier            = "code_verifier"
	CcmPlain                = "plain"
	CcmS256                 = "S256"
)

func NewOAuthAuthorizationUrlFunction(oauth2Configs []config.OAuth2Authorization, verifier func() (*pkce.CodeVerifier, error)) function.Function {
	oauth2s := make(map[string]config.OAuth2Authorization)
	for _, o := range oauth2Configs {
		oauth2s[o.GetName()] = o
	}
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "oauth2_label",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			label := args[0].AsString()
			oauth2 := oauth2s[label]

			authorizationEndpoint, err := oauth2.GetAuthorizationEndpoint()
			if err != nil {
				return cty.StringVal(""), err
			}

			oauthAuthorizationUrl, err := url.Parse(authorizationEndpoint)
			if err != nil {
				return cty.StringVal(""), err
			}

			query := oauthAuthorizationUrl.Query()
			query.Set("response_type", "code")
			query.Set("client_id", oauth2.GetClientID())
			query.Set("redirect_uri", *oauth2.GetRedirectURI())
			if scope := oauth2.GetScope(); scope != nil {
				query.Set("scope", *scope)
			}

			if pkce := oauth2.GetPkce(); pkce != nil && pkce.CodeChallengeMethod != "" {
				query.Set("code_challenge_method", pkce.CodeChallengeMethod)
				codeChallenge, err := createCodeChallenge(verifier, pkce.CodeChallengeMethod)
				if err != nil {
					return cty.StringVal(""), err
				}

				query.Set("code_challenge", codeChallenge)
			} else if csrf := oauth2.GetCsrf(); csrf != nil && (csrf.TokenParam == "state" || csrf.TokenParam == "nonce") {
				hashedCsrfToken, err := createCodeChallenge(verifier, CcmS256)
				if err != nil {
					return cty.StringVal(""), err
				}

				query.Set(csrf.TokenParam, hashedCsrfToken)
			}
			oauthAuthorizationUrl.RawQuery = query.Encode()

			return cty.StringVal(oauthAuthorizationUrl.String()), nil
		},
	})
}

func NewOAuthCodeVerifierFunction(verifier func() (*pkce.CodeVerifier, error)) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			codeVerifier, err := verifier()
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(codeVerifier.String()), nil
		},
	})
}

func NewOAuthCodeChallengeFunction(verifier func() (*pkce.CodeVerifier, error)) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "code_challenge_method",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			method := args[0].AsString()
			codeChallenge, err := createCodeChallenge(verifier, method)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(codeChallenge), nil
		},
	})
}

func NewOAuthHashedCsrfTokenFunction(verifier func() (*pkce.CodeVerifier, error)) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			hashedCsrfToken, err := createCodeChallenge(verifier, CcmS256)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(hashedCsrfToken), nil
		},
	})
}

func createCodeChallenge(verifier func() (*pkce.CodeVerifier, error), method string) (string, error) {
	codeVerifier, err := verifier()
	if err != nil {
		return "", err
	}

	switch method {
	case CcmS256:
		return codeVerifier.CodeChallengeS256(), nil
	case CcmPlain:
		return codeVerifier.CodeChallengePlain(), nil
	default:
		return "", fmt.Errorf("unsupported code challenge method: %s", method)
	}
}
