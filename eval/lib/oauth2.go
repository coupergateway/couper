package lib

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/hcl/v2"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/internal/seetie"
)

const (
	RedirectURI                   = "redirect_uri"
	CodeVerifier                  = "code_verifier"
	FnOAuthAuthorizationUrl       = "oauth2_authorization_url"
	FnOAuthVerifier               = "oauth2_verifier"
	InternalFnOAuthHashedVerifier = "internal_oauth_hashed_verifier"
)

func NewOAuthAuthorizationUrlFunction(ctx *hcl.EvalContext, oauth2Configs []config.OAuth2Authorization,
	verifier func() (*pkce.CodeVerifier, error), origin *url.URL,
	evalFn func(*hcl.EvalContext, hcl.Expression) (cty.Value, error)) function.Function {
	oauth2s := make(map[string]config.OAuth2Authorization)
	for _, o := range oauth2Configs {
		oauth2s[o.GetName()] = o
	}

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
				return emptyStringVal, fmt.Errorf("undefined reference: %s", label)
			}

			uid := seetie.ToString(seetie.ValueToMap(ctx.Variables["request"])["id"])
			authorizationEndpoint, err := oauth2.GetAuthorizationEndpoint(uid)
			if err != nil {
				return emptyStringVal, err
			}

			oauthAuthorizationUrl, err := url.Parse(authorizationEndpoint)
			if err != nil {
				return emptyStringVal, err
			}

			body := oauth2.HCLBody()
			bodyContent, _, diags := body.PartialContent(oauth2.Schema(true))
			if diags.HasErrors() {
				return emptyStringVal, diags
			}

			var redirectURI string
			if attr, ok := bodyContent.Attributes[RedirectURI]; ok {
				val, verr := evalFn(ctx, attr.Expr)
				if verr != nil {
					return emptyStringVal, verr
				}
				redirectURI = seetie.ValueToString(val)
			}

			if redirectURI == "" {
				return emptyStringVal, fmt.Errorf("%s is required", RedirectURI)
			}

			absRedirectUri, err := AbsoluteURL(redirectURI, origin)
			if err != nil {
				return emptyStringVal, err
			}

			query := oauthAuthorizationUrl.Query()
			query.Set("response_type", "code")
			query.Set("client_id", oauth2.GetClientID())
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

func createCodeChallenge(verifier func() (*pkce.CodeVerifier, error)) (string, error) {
	codeVerifier, err := verifier()
	if err != nil {
		return "", err
	}

	return codeVerifier.CodeChallengeS256(), nil
}
