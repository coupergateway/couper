package lib

import (
	"crypto/rsa"
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
	"github.com/avenga/couper/internal/seetie"
)

const FnJWTSign = "jwt_sign"

var rsaParseError = &rsa.PrivateKey{}

func NewJwtSignFunction(jwtSigningProfiles []*config.JWTSigningProfile, confCtx *hcl.EvalContext) function.Function {
	signingProfiles := make(map[string]*config.JWTSigningProfile)
	rsaKeys := make(map[string]*rsa.PrivateKey)

	for _, sp := range jwtSigningProfiles {
		signingProfiles[sp.Name] = sp
		if strings.HasPrefix(sp.SignatureAlgorithm, "RS") {
			key, err := jwt.ParseRSAPrivateKeyFromPEM(sp.KeyBytes)
			if err != nil {
				rsaKeys[sp.Name] = rsaParseError
				continue
			}
			rsaKeys[sp.Name] = key
		}
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
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile definitions")
			}

			label := args[0].AsString()
			signingProfile := signingProfiles[label]
			if signingProfile == nil {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile for given label: %s", label)
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
				ttl, parseErr := time.ParseDuration(signingProfile.TTL)
				if parseErr != nil {
					return cty.StringVal(""), parseErr
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

			var key interface{}
			if rsaKey, exist := rsaKeys[signingProfile.Name]; exist {
				if rsaKey == rsaParseError {
					return cty.StringVal(""), fmt.Errorf("could not parse rsa private key from pem: %s", signingProfile.Name)
				}
				key = rsaKey
			} else {
				key = signingProfile.KeyBytes
			}

			tokenString, err := CreateJWT(signingProfile.SignatureAlgorithm, key, mapClaims)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(tokenString), nil
		},
	})
}

func CreateJWT(signatureAlgorithm string, key interface{}, mapClaims jwt.MapClaims) (string, error) {
	signingMethod := jwt.GetSigningMethod(signatureAlgorithm)
	if signingMethod == nil {
		return "", fmt.Errorf("no signing method for given algorithm: %s", signatureAlgorithm)
	}

	// create token
	token := jwt.NewWithClaims(signingMethod, mapClaims)

	// sign token
	return token.SignedString(key)
}
