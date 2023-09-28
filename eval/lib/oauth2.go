package lib

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/hcl/v2"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/coupergateway/couper/config"
)

const (
	CodeVerifier                  = "code_verifier"
	FnOAuthAuthorizationURL       = "oauth2_authorization_url"
	FnOAuthVerifier               = "oauth2_verifier"
	InternalFnOAuthHashedVerifier = "internal_oauth_hashed_verifier"
)

var NoOpOAuthAuthorizationURLFunction = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "oauth2_label",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
		if len(args) > 0 {
			return cty.StringVal(""), fmt.Errorf("missing oidc or beta_oauth2 block with referenced label %q", args[0].AsString())
		}
		return cty.StringVal(""), fmt.Errorf("missing oidc or beta_oauth2 definitions")
	},
})

func NewOAuthAuthorizationURLFunction(ctx *hcl.EvalContext, oauth2s map[string]config.OAuth2Authorization,
	verifier func() (*pkce.CodeVerifier, error), origin *url.URL,
	evalFn func(*hcl.EvalContext, hcl.Expression) (cty.Value, error)) function.Function {

	emptyStringVal := cty.StringVal("")

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
				return NoOpOAuthAuthorizationURLFunction.Call(args)
			}

			authorizationEndpoint, err := oauth2.GetAuthorizationEndpoint()
			if err != nil {
				return emptyStringVal, err
			}

			oauthAuthorizationURL, err := url.Parse(authorizationEndpoint)
			if err != nil {
				return emptyStringVal, err
			}

			redirectURI := oauth2.GetRedirectURI()
			if redirectURI == "" {
				return emptyStringVal, fmt.Errorf("redirect_uri is required")
			}

			absRedirectURI, err := AbsoluteURL(redirectURI, origin)
			if err != nil {
				return emptyStringVal, err
			}

			query := oauthAuthorizationURL.Query()
			query.Set("response_type", "code")
			query.Set("client_id", oauth2.GetClientID())
			query.Set("redirect_uri", absRedirectURI)
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
			oauthAuthorizationURL.RawQuery = query.Encode()

			return cty.StringVal(oauthAuthorizationURL.String()), nil
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

func createCodeChallenge(verifier func() (*pkce.CodeVerifier, error)) (string, error) {
	codeVerifier, err := verifier()
	if err != nil {
		return "", err
	}

	return codeVerifier.CodeChallengeS256(), nil
}
