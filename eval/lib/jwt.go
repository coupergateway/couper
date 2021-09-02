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

type JWTSigningConfig struct {
	Claims             config.Claims
	KeyBytes           []byte
	Name               string
	SignatureAlgorithm string
	TTL                string
}

func NewJWTSigningConfigFromJWTSigningProfile(j *config.JWTSigningProfile) *JWTSigningConfig {
	c := &JWTSigningConfig{
		Claims:             j.Claims,
		KeyBytes:           j.KeyBytes,
		Name:               j.Name,
		SignatureAlgorithm: j.SignatureAlgorithm,
		TTL:                j.TTL,
	}
	return c
}

func NewJwtSignFunction(jwtSigningConfigs []*JWTSigningConfig, confCtx *hcl.EvalContext) function.Function {
	signingConfigs := make(map[string]*JWTSigningConfig)
	rsaKeys := make(map[string]*rsa.PrivateKey)

	for _, jsc := range jwtSigningConfigs {
		signingConfigs[jsc.Name] = jsc
		if strings.HasPrefix(jsc.SignatureAlgorithm, "RS") {
			key, err := jwt.ParseRSAPrivateKeyFromPEM(jsc.KeyBytes)
			if err != nil {
				rsaKeys[jsc.Name] = rsaParseError
				continue
			}
			rsaKeys[jsc.Name] = key
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
			if len(signingConfigs) == 0 {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile definitions")
			}

			label := args[0].AsString()
			signingConfig := signingConfigs[label]
			if signingConfig == nil {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile for given label: %s", label)
			}

			mapClaims := jwt.MapClaims{}
			var defaultClaims, argumentClaims map[string]interface{}

			// get claims from signing profile
			if signingConfig.Claims != nil {
				c, diags := seetie.ExpToMap(confCtx, signingConfig.Claims)
				if diags.HasErrors() {
					return cty.StringVal(""), diags
				}
				defaultClaims = c
			}

			for k, v := range defaultClaims {
				mapClaims[k] = v
			}
			if signingConfig.TTL != "0" {
				ttl, parseErr := time.ParseDuration(signingConfig.TTL)
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
			if rsaKey, exist := rsaKeys[signingConfig.Name]; exist {
				if rsaKey == rsaParseError {
					return cty.StringVal(""), fmt.Errorf("could not parse rsa private key from pem: %s", signingConfig.Name)
				}
				key = rsaKey
			} else {
				key = signingConfig.KeyBytes
			}

			tokenString, err := CreateJWT(signingConfig.SignatureAlgorithm, key, mapClaims)
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
