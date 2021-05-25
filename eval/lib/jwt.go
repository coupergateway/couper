package lib

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/avenga/couper/config"
	couperErr "github.com/avenga/couper/errors"
	"github.com/avenga/couper/internal/seetie"
)

const FnJWTSign = "jwt_sign"

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
			if len(signingProfiles) == 0 {
				return cty.StringVal(""), fmt.Errorf("no jwt_signing_profile definitions found")
			}

			label := args[0].AsString()
			signingProfile := signingProfiles[label]
			if signingProfile == nil {
				return cty.StringVal(""), couperErr.NewJWTError(couperErr.ErrorNoProfileForLabel)
			}

			keyData, err := couperErr.ValidateJWTKey(
				signingProfile.SignatureAlgorithm, signingProfile.Key, signingProfile.KeyFile,
			)
			if err != nil {
				return cty.StringVal(""), err
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
			signingMethod := jwt.GetSigningMethod(signingProfile.SignatureAlgorithm)
			if signingMethod == nil {
				return cty.StringVal(""), couperErr.NewJWTError(couperErr.ErrorUnsupportedSigningMethod)
			}

			token := jwt.NewWithClaims(signingMethod, mapClaims)

			var key interface{}
			if strings.HasPrefix(signingProfile.SignatureAlgorithm, "RS") {
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
