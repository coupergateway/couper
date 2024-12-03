package test

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
)

func RSAPubKeyToJWK(key rsa.PublicKey) map[string]interface{} {
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	return map[string]interface{}{
		"kty": "RSA",
		"n":   n,
		"e":   e,
	}
}
