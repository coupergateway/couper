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
	Key                interface{}
	Name               string
	SignatureAlgorithm string
	TTL                time.Duration
}

func NewJWTSigningConfigFromJWTSigningProfile(j *config.JWTSigningProfile) (*JWTSigningConfig, error) {
	if alg := acjwt.NewAlgorithm(j.SignatureAlgorithm); alg == acjwt.AlgorithmUnknown {
		return nil, fmt.Errorf("algorithm is not supported")
	}

	var (
		ttl      time.Duration
		parseErr error
	)
	if j.TTL != "0" {
		ttl, parseErr = time.ParseDuration(j.TTL)
		if parseErr != nil {
			return nil, parseErr
		}
	}

	var key interface{}
	key = j.KeyBytes
	if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
		key, parseErr = jwt.ParseRSAPrivateKeyFromPEM(j.KeyBytes)
		if parseErr != nil {
			return nil, parseErr
		}
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

func NewJWTSigningConfigFromJWT(j *config.JWT) (*JWTSigningConfig, error) {
	if j.SigningTTL == "" {
		return nil, nil
	}

	alg := acjwt.NewAlgorithm(j.SignatureAlgorithm)
	if alg == acjwt.AlgorithmUnknown {
		return nil, fmt.Errorf("algorithm is not supported")
	}

	var (
		ttl      time.Duration
		parseErr error
	)
	if j.SigningTTL != "0" {
		ttl, parseErr = time.ParseDuration(j.SigningTTL)
		if parseErr != nil {
			return nil, parseErr
		}
	}

	var (
		signingKey, signingKeyFile string
	)

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

	var key interface{}
	key = keyBytes
	if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
		key, parseErr = jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
		if parseErr != nil {
			return nil, parseErr
		}
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

func NewJwtSignFunction(jwtSigningConfigs []*JWTSigningConfig, confCtx *hcl.EvalContext) function.Function {
	signingConfigs := make(map[string]*JWTSigningConfig)
	for _, jsc := range jwtSigningConfigs {
		signingConfigs[jsc.Name] = jsc
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
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt definitions")
			}

			label := args[0].AsString()
			signingConfig := signingConfigs[label]
			if signingConfig == nil {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt for given label: %s", label)
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

			tokenString, err := CreateJWT(signingConfig.SignatureAlgorithm, signingConfig.Key, mapClaims)
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
