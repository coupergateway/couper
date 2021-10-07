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

	acjwt "github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/internal/seetie"
)

const FnJWTSign = "jwt_sign"

var rsaParseError = &rsa.PrivateKey{}

type JWTSigningConfig struct {
	Claims             config.Claims
	Headers            hcl.Expression
	Key                interface{}
	Name               string
	SignatureAlgorithm string
	TTL                time.Duration
}

func checkData(ttl, signatureAlgorithm string) (time.Duration, acjwt.Algorithm, error) {
	var (
		dur      time.Duration
		parseErr error
	)

	alg := acjwt.NewAlgorithm(signatureAlgorithm)
	if alg == acjwt.AlgorithmUnknown {
		return dur, alg, fmt.Errorf("algorithm is not supported")
	}

	if ttl != "0" {
		dur, parseErr = time.ParseDuration(ttl)
		if parseErr != nil {
			return dur, alg, parseErr
		}
	}

	return dur, alg, nil
}

func getKey(keyBytes []byte, signatureAlgorithm string) (interface{}, error) {
	var (
		key      interface{}
		parseErr error
	)
	key = keyBytes
	if strings.HasPrefix(signatureAlgorithm, "RS") {
		key, parseErr = jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return key, nil
}

func NewJWTSigningConfigFromJWTSigningProfile(j *config.JWTSigningProfile) (*JWTSigningConfig, error) {
	ttl, _, err := checkData(j.TTL, j.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	key, err := getKey(j.KeyBytes, j.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	c := &JWTSigningConfig{
		Claims:             j.Claims,
		Headers:            j.Headers,
		Key:                key,
		Name:               j.Name,
		SignatureAlgorithm: j.SignatureAlgorithm,
		TTL:                ttl,
	}
	return c, nil
}

func NewJWTSigningConfigFromJWT(j *config.JWT) (*JWTSigningConfig, error) {
	if j.SigningTTL == "" {
		return nil, nil
	}

	ttl, alg, err := checkData(j.SigningTTL, j.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	var signingKey, signingKeyFile string

	if alg.IsHMAC() {
		signingKey = j.Key
		signingKeyFile = j.KeyFile
	} else {
		signingKey = j.SigningKey
		signingKeyFile = j.SigningKeyFile
	}
	keyBytes, err := reader.ReadFromAttrFile("jwt signing key", signingKey, signingKeyFile)
	if err != nil {
		return nil, err
	}

	key, err := getKey(keyBytes, j.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	c := &JWTSigningConfig{
		Claims:             j.Claims,
		Key:                key,
		Name:               j.Name,
		SignatureAlgorithm: j.SignatureAlgorithm,
		TTL:                ttl,
	}
	return c, nil
}

func NewJwtSignFunction(jwtSigningConfigs map[string]*JWTSigningConfig, confCtx *hcl.EvalContext) function.Function {
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
			if len(jwtSigningConfigs) == 0 {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt definitions")
			}

			label := args[0].AsString()
			signingConfig := jwtSigningConfigs[label]
			if signingConfig == nil {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt for given label: %s", label)
			}

			mapClaims := jwt.MapClaims{}
			var defaultClaims, argumentClaims, headers map[string]interface{}

			if signingConfig.Headers != nil {
				h, diags := seetie.ExpToMap(confCtx, signingConfig.Headers)
				if diags.HasErrors() {
					return cty.StringVal(""), diags
				}
				headers = h
			}

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
			if signingConfig.TTL != 0 {
				mapClaims["exp"] = time.Now().Unix() + int64(signingConfig.TTL.Seconds())
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

			tokenString, err := CreateJWT(signingConfig.SignatureAlgorithm, signingConfig.Key, mapClaims, headers)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(tokenString), nil
		},
	})
}

func CreateJWT(signatureAlgorithm string, key interface{}, mapClaims jwt.MapClaims, headers map[string]interface{}) (string, error) {
	signingMethod := jwt.GetSigningMethod(signatureAlgorithm)
	if signingMethod == nil {
		return "", fmt.Errorf("no signing method for given algorithm: %s", signatureAlgorithm)
	}

	if headers == nil {
		headers = map[string]interface{}{}
	}

	headers["typ"] = "JWT"
	headers["alg"] = signingMethod.Alg()

	// create token
	token := &jwt.Token{Header: headers, Claims: mapClaims, Method: signingMethod}

	// sign token
	return token.SignedString(key)
}
