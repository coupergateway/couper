package eval

import (
	"context"
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

func NewOAuthCodeVerifierFunction(ctx *Context) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			codeVerifier, err := getCodeVerifier(ctx)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(codeVerifier.String()), nil
		},
	})
}

func NewOAuthCodeChallengeFunction(ctx *Context) function.Function {
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
			codeVerifier, err := getCodeVerifier(ctx)
			if err != nil {
				return cty.StringVal(""), err
			}

			switch method {
			case CCM_S256:
				return cty.StringVal(codeVerifier.CodeChallengeS256()), nil
			case CCM_plain:
				return cty.StringVal(codeVerifier.CodeChallengePlain()), nil
			default:
				return cty.StringVal(""), fmt.Errorf("Unsupported code challenge method: %s", method)
			}
		},
	})
}

func getCodeVerifier(ctx *Context) (*pkce.CodeVerifier, error) {
	codeVerifier, ok := ctx.Value(CodeVerifier).(*pkce.CodeVerifier)
	if !ok {
		var err error
		if codeVerifier, err = pkce.CreateCodeVerifier(); err != nil {
			return nil, err
		}

		ctx.inner = context.WithValue(ctx.inner, CodeVerifier, codeVerifier)
	}
	return codeVerifier, nil
}
