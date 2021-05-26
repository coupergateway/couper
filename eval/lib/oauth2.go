package lib

import (
	"fmt"

	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

const (
	FnOAuthCodeVerifier  = "oauth_code_verifier"
	FnOAuthCodeChallenge = "oauth_code_challenge"
	CodeVerifier         = "code_verifier"
	CCM_plain            = "plain"
	CCM_S256             = "S256"
)

func NewOAuthCodeVerifierFunction(verifier interface{}) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			codeVerifier, ok := verifier.(*pkce.CodeVerifier)
			if !ok {
				return cty.StringVal(""), err
			}

			return cty.StringVal(codeVerifier.String()), nil
		},
	})
}

func NewOAuthCodeChallengeFunction(verifier interface{}) function.Function {
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
			codeVerifier, ok := verifier.(*pkce.CodeVerifier)
			if !ok {
				return cty.StringVal(""), err
			}

			switch method {
			case CCM_S256:
				return cty.StringVal(codeVerifier.CodeChallengeS256()), nil
			case CCM_plain:
				return cty.StringVal(codeVerifier.CodeChallengePlain()), nil
			default:
				return cty.StringVal(""), fmt.Errorf("unsupported code challenge method: %s", method)
			}
		},
	})
}
