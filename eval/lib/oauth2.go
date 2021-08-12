package lib

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/hcl/v2"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/eval/content"
)

const (
	CallbackURL                   = "redirect_uri"
	CodeVerifier                  = "code_verifier"
	FnOAuthAuthorizationUrl       = "beta_oauth_authorization_url"
	FnOAuthVerifier               = "beta_oauth_verifier"
	InternalFnOAuthHashedVerifier = "internal_oauth_hashed_verifier"
)

func NewOAuthAuthorizationUrlFunction(ctx *hcl.EvalContext, oauth2Configs []config.OAuth2Authorization,
	verifier func() (*pkce.CodeVerifier, error), origin *url.URL) function.Function {
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
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			label := args[0].AsString()
			oauth2, exist := oauth2s[label]
			if !exist {
				return cty.StringVal(""), fmt.Errorf("undefined reference: %s", label)
			}

			authorizationEndpoint, err := oauth2.GetAuthorizationEndpoint()
			if err != nil {
				return cty.StringVal(""), err
			}

			oauthAuthorizationUrl, err := url.Parse(authorizationEndpoint)
			if err != nil {
				return cty.StringVal(""), err
			}

			var callbackURL string
			if inline, ok := oauth2.(config.Inline); ok {
				body := inline.HCLBody()
				bodyContent, _, diags := body.PartialContent(inline.Schema(true))
				if diags.HasErrors() {
					return cty.StringVal(""), diags
				}
				val, err := content.GetAttribute(ctx, bodyContent, CallbackURL)
				if err != nil {
					return cty.StringVal(""), err
				} else if val == "" {
					return cty.StringVal(""), fmt.Errorf("%s is required", CallbackURL)
				}

				callbackURL = val
			}

			query := oauthAuthorizationUrl.Query()
			query.Set("response_type", "code")
			query.Set("client_id", oauth2.GetClientID())
			absRedirectUri, err := AbsoluteURL(callbackURL, origin)
			if err != nil {
				return cty.StringVal(""), err
			}
			query.Set("redirect_uri", absRedirectUri)
			if scope := oauth2.GetScope(); scope != "" {
				query.Set("scope", scope)
			}

			verifierMethod, err := oauth2.GetVerifierMethod()
			if err != nil {
				return cty.StringVal(""), err
			}

			if verifierMethod == config.CcmS256 {
				codeChallenge, err := createCodeChallenge(verifier)
				if err != nil {
					return cty.StringVal(""), err
				}

				query.Set("code_challenge_method", "S256")
				query.Set("code_challenge", codeChallenge)
			} else {
				hashedVerifier, err := createCodeChallenge(verifier)
				if err != nil {
					return cty.StringVal(""), err
				}

				query.Set(verifierMethod, hashedVerifier)
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
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			codeChallenge, err := createCodeChallenge(verifier)
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
			hashedVerifier, err := createCodeChallenge(verifier)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(hashedVerifier), nil
		},
	})
}

func createCodeChallenge(verifier func() (*pkce.CodeVerifier, error)) (string, error) {
	codeVerifier, err := verifier()
	if err != nil {
		return "", err
	}

	return codeVerifier.CodeChallengeS256(), nil
}
