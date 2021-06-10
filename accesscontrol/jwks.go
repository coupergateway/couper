package accesscontrol

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/avenga/couper/utils/base64url"
)

type JWKS struct {
	Keys []JWK `json:"keys"`
}

// a JWK Set only "SHOULD" use distinct key IDs
func (j JWKS) Key(kid string) []JWK {
	var keys []JWK
	for _, key := range j.Keys {
		if key.KeyID == kid {
			keys = append(keys, key)
		}
	}
	return keys
}

type rawJWK struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	// RSA public key
	E *keyParam `json:"e"`
	N *keyParam `json:"n"`
}

type JWK struct {
	Key       interface{}
	KeyID     string
	Algorithm string
	Use       string
}

func (j JWK) MarshalJSON() ([]byte, error) {
	var raw *rawJWK
	switch key := j.Key.(type) {
	case *rsa.PublicKey:
		raw = fromRsaPublicKey(key)
	default:
		return nil, fmt.Errorf("kty '%s' not supported", reflect.TypeOf(key))
	}
	raw.Kid = j.KeyID
	raw.Alg = j.Algorithm
	raw.Use = j.Use

	return json.Marshal(raw)
}

func (j *JWK) UnmarshalJSON(data []byte) error {
	var raw rawJWK
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	var key interface{}

	switch raw.Kty {
	case "RSA":
		key = &rsa.PublicKey{
			N: raw.N.toBigInt(),
			E: raw.E.toInt(),
		}
	default:
		return fmt.Errorf("kty '%s' not supported", raw.Kty)
	}

	*j = JWK{Key: key, KeyID: raw.Kid, Algorithm: raw.Alg, Use: raw.Use}

	return nil
}

func fromRsaPublicKey(pub *rsa.PublicKey) *rawJWK {
	return &rawJWK{
		Kty: "RSA",
		N:   newKeyParam(pub.N.Bytes()),
		E:   newKeyParamFromInt(uint64(pub.E)),
	}
}

type keyParam struct {
	data []byte
}

func newKeyParam(data []byte) *keyParam {
	if data == nil {
		return nil
	}
	return &keyParam{
		data: data,
	}
}

func newKeyParamFromInt(num uint64) *keyParam {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, num)
	return newKeyParam(bytes.TrimLeft(data, "\x00"))
}

func (k *keyParam) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64url.Encode(k.data))
}

func (k *keyParam) UnmarshalJSON(data []byte) error {
	var encoded string
	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return err
	}

	if encoded == "" {
		return nil
	}

	decoded, err := base64url.Decode(encoded)
	if err != nil {
		return err
	}

	*k = *newKeyParam(decoded)

	return nil
}

func (k keyParam) toBigInt() *big.Int {
	return new(big.Int).SetBytes(k.data)
}

func (k keyParam) toInt() int {
	return int(k.toBigInt().Int64())
}
