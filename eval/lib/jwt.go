package lib

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/internal/seetie"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

var (
	ErrorNoProfileForLabel        = errors.New("no signing profile for label")
	ErrorMissingKey               = errors.New("either key_file or key must be specified")
	ErrorUnsupportedSigningMethod = errors.New("unsupported signing method")
)

type JwtSigningError struct {
	error
}

func (e *JwtSigningError) Error() string {
	return e.error.Error()
}

func NewJwtSignFunction(jwtSigningProfiles []*config.JWTSigningProfile, confCtx *hcl.EvalContext) function.Function {
	signingProfiles := make(map[string]*config.JWTSigningProfile)
	for _, sp := range jwtSigningProfiles {
		signingProfiles[sp.Name] = sp
	}
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "jwt_signing_profile_label",
				Type: cty.String,
			},
			{
				Name: "claims",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			label := args[0].AsString()
			signingProfile := signingProfiles[label]
			if signingProfile == nil {
				return cty.StringVal(""), &JwtSigningError{error: ErrorNoProfileForLabel}
			}

			// get key or secret
			var keyData []byte
			if signingProfile.KeyFile != "" {
				p, err := filepath.Abs(signingProfile.KeyFile)
				if err != nil {
					return cty.StringVal(""), err
				}
				content, err := ioutil.ReadFile(p)
				if err != nil {
					return cty.StringVal(""), err
				}
				keyData = content
			} else if signingProfile.Key != "" {
				keyData = []byte(signingProfile.Key)
			}
			if len(keyData) == 0 {
				return cty.StringVal(""), &JwtSigningError{error: ErrorMissingKey}
			}

			mapClaims := jwt.MapClaims{}
			var defaultClaims, argumentClaims map[string]interface{}

			// get claims from signing profile
			if signingProfile.Claims != nil {
				c, diags := seetie.ExpToMap(confCtx, signingProfile.Claims)
				if diags.HasErrors() {
					return cty.StringVal(""), diags
				}
				defaultClaims = c
			}

			for k, v := range defaultClaims {
				mapClaims[k] = v
			}
			if signingProfile.TTL != "0" {
				ttl, err := time.ParseDuration(signingProfile.TTL)
				if err != nil {
					return cty.StringVal(""), err
				}
				mapClaims["exp"] = time.Now().Unix() + int64(ttl.Seconds())
			}

			// get claims from function argument
			jsonClaims, err := stdlib.JSONEncode(args[1])
			if err != nil {
				return cty.StringVal(""), err
			}

			err = json.Unmarshal([]byte(jsonClaims.AsString()), &argumentClaims)
			if err != nil {
				return cty.StringVal(""), err
			}

			for k, v := range argumentClaims {
				mapClaims[k] = v
			}

			// create token
			signingMethod := jwt.GetSigningMethod(signingProfile.SignatureAlgorithm);
			if signingMethod == nil {
				return cty.StringVal(""), &JwtSigningError{error: ErrorUnsupportedSigningMethod}
			}

			token := jwt.NewWithClaims(signingMethod, mapClaims)

			var key interface{}
			if (strings.HasPrefix(signingProfile.SignatureAlgorithm, "RS")) {
				key, err = jwt.ParseRSAPrivateKeyFromPEM(keyData)
				if err != nil {
					return cty.StringVal(""), err
				}
			} else {
				key = keyData
			}

			// sign token
			tokenString, err := token.SignedString(key)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(tokenString), nil
		},
	})
}
