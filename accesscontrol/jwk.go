package accesscontrol

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
)

type JWK struct {
	Key       interface{}
	KeyID     string
	Algorithm string
	Use       string
}

type rawJWK struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	// RSA public key
	E   *base64URLEncodedField `json:"e"`
	N   *base64URLEncodedField `json:"n"`
	X5c []*base64EncodedField  `json:"x5c"`
	//X5t *base64URLEncodedField `json:"x5t"`
	//X5tS256" *base64URLEncodedField `json:"x5t#S256"`
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

func (self *JWK) UnmarshalJSON(data []byte) error {
	var raw rawJWK
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	var key interface{}

	switch raw.Kty {
	case "RSA":
		if len(raw.X5c) > 0 {
			certificate, err := x509.ParseCertificate(raw.X5c[0].data)
			if err != nil {
				return fmt.Errorf("Invalid x5c: %v", err)
			}

			key = certificate.PublicKey.(*rsa.PublicKey)
		} else if raw.N != nil && raw.E != nil {
			key = &rsa.PublicKey{
				N: raw.N.toBigInt(),
				E: raw.E.toInt(),
			}
		} else {
			fmt.Printf("Ignoring invalid %s key: %q", raw.Kty, raw.Kid)
			return nil
		}

	default:
		fmt.Printf("Found unsupported %s key: %q\n", raw.Kty, raw.Kid)
	}

	*self = JWK{Key: key, KeyID: raw.Kid, Algorithm: raw.Alg, Use: raw.Use}

	return nil
}

func fromRsaPublicKey(pub *rsa.PublicKey) *rawJWK {
	return &rawJWK{
		Kty: "RSA",
		N:   newBase64URLEncodedField(pub.N.Bytes()),
		E:   newBase64EncodedFieldFromInt(uint64(pub.E)),
	}
}

// Base64URL encoded

type base64URLEncodedField struct {
	data []byte
}

func newBase64URLEncodedField(data []byte) *base64URLEncodedField {
	return &base64URLEncodedField{
		data: data,
	}
}

func newBase64EncodedFieldFromInt(num uint64) *base64URLEncodedField {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, num)
	return newBase64URLEncodedField(bytes.TrimLeft(data, "\x00"))
}

func (self *base64URLEncodedField) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.RawURLEncoding.EncodeToString(self.data))
}

func (self *base64URLEncodedField) UnmarshalJSON(data []byte) error {
	var encoded string
	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return err
	}

	if encoded == "" {
		return nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	*self = *newBase64URLEncodedField(decoded)

	return nil
}

func (self base64URLEncodedField) toBigInt() *big.Int {
	return new(big.Int).SetBytes(self.data)
}

func (self base64URLEncodedField) toInt() int {
	return int(self.toBigInt().Int64())
}

// Base64 encoded

type base64EncodedField struct {
	data []byte
}

func (self *base64EncodedField) UnmarshalJSON(data []byte) error {
	var encoded string
	err := json.Unmarshal(data, &encoded)
	if err != nil {
		return err
	}

	if encoded == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	*self = base64EncodedField{
		data: decoded,
	}

	return nil
}
