package lib

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

type JWTSigningConfig struct {
	Claims             config.Claims
	Headers            hcl.Expression
	Key                interface{}
	Name               string
	SignatureAlgorithm string
	TTL                time.Duration
}

func CheckData(ttl, signatureAlgorithm string) (time.Duration, acjwt.Algorithm, error) {
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

func GetKey(keyBytes []byte, signatureAlgorithm string) (interface{}, error) {
	var (
		key      interface{}
		parseErr error
	)
	key = keyBytes
	if strings.HasPrefix(signatureAlgorithm, "RS") {
		key, parseErr = jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	} else if strings.HasPrefix(signatureAlgorithm, "ES") {
		key, parseErr = parseECPrivateKeyFromPEM(keyBytes)
	}

	return key, parseErr
}

func NewJWTSigningConfigFromJWTSigningProfile(j *config.JWTSigningProfile) (*JWTSigningConfig, error) {
	ttl, _, err := CheckData(j.TTL, j.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	key, err := GetKey(j.KeyBytes, j.SignatureAlgorithm)
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

	ttl, alg, err := CheckData(j.SigningTTL, j.SignatureAlgorithm)
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

	key, err := GetKey(keyBytes, j.SignatureAlgorithm)
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

func NewJwtSignFunction(ctx *hcl.EvalContext, jwtSigningConfigs map[string]*JWTSigningConfig,
	evalFn func(*hcl.EvalContext, hcl.Expression) (cty.Value, error)) function.Function {
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
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt (with signing_ttl) definitions")
			}

			label := args[0].AsString()
			signingConfig := jwtSigningConfigs[label]
			if signingConfig == nil {
				return cty.StringVal(""), fmt.Errorf("missing jwt_signing_profile or jwt (with signing_ttl) for given label %q", label)
			}

			mapClaims := jwt.MapClaims{}
			var defaultClaims, argumentClaims, headers map[string]interface{}

			if signingConfig.Headers != nil {
				h, diags := evalFn(ctx, signingConfig.Headers)
				if diags != nil {
					return cty.StringVal(""), diags
				}
				headers = seetie.ValueToMap(h)
			}

			// get claims from signing profile
			if signingConfig.Claims != nil {
				v, diags := evalFn(ctx, signingConfig.Claims)
				if diags != nil {
					return cty.StringVal(""), err
				}
				defaultClaims = seetie.ValueToMap(v)
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

func parseECPrivateKeyFromPEM(key []byte) (*ecdsa.PrivateKey, error) {
	var err error

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, jwt.ErrKeyMustBePEMEncoded
	}

	// Parse the key
	var parsedKey interface{}
	if parsedKey, err = x509.ParseECPrivateKey(block.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
			return nil, err
		}
	}

	var pkey *ecdsa.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(*ecdsa.PrivateKey); !ok {
		return nil, jwt.ErrNotECPrivateKey
	}

	return pkey, nil
}
