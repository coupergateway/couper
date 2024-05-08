package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

type JWK struct {
	Key       interface{}
	KeyID     string
	KeyType   string
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
	// ECDSA public key
	Crv string                 `json:"crv"`
	X   *base64URLEncodedField `json:"x"`
	Y   *base64URLEncodedField `json:"y"`
	//X5t *base64URLEncodedField `json:"x5t"`
	//X5tS256" *base64URLEncodedField `json:"x5t#S256"`
}

func (j *JWK) UnmarshalJSON(data []byte) error {
	raw := &rawJWK{}
	err := json.Unmarshal(data, raw)
	if err != nil {
		// TODO log warning properly
		fmt.Printf("Invalid JWK: %v\n", err)
		return nil
	}

	var key interface{}
	jwk := &JWK{KeyID: raw.Kid, Algorithm: raw.Alg, KeyType: raw.Kty, Use: raw.Use}

	switch raw.Kty {
	case "RSA":
		key, err = getPublicKeyFromX5c(raw.X5c)
		if err != nil {
			// TODO log warning properly
			fmt.Printf("Invalid x5c: %v\n", err)
			return nil
		}

		if key != nil {
			jwk.Key = key
		} else if raw.N != nil && raw.E != nil {
			jwk.Key = &rsa.PublicKey{
				N: raw.N.toBigInt(),
				E: raw.E.toInt(),
			}
		} else {
			// TODO log warning properly
			fmt.Printf("Ignoring invalid %s key: %q\n", raw.Kty, raw.Kid)
			return nil
		}
	case "EC":
		key, err = getPublicKeyFromX5c(raw.X5c)
		if err != nil {
			// TODO log warning properly
			fmt.Printf("Invalid x5c: %v\n", err)
			return nil
		}

		if key != nil {
			jwk.Key = key
		} else {
			curve, err := GetCurve(raw.Crv)
			if err == nil && raw.X != nil && raw.Y != nil {
				jwk.Key = &ecdsa.PublicKey{
					Curve: curve,
					X:     raw.X.toBigInt(),
					Y:     raw.Y.toBigInt(),
				}
			} else {
				fmt.Printf("Ignoring invalid %s key: %q (invalid crv/x/y)\n", raw.Kty, raw.Kid)
				return nil
			}
		}
	default:
		// TODO log warning properly
		fmt.Printf("Found unsupported %s key: %q\n", raw.Kty, raw.Kid)
		return nil
	}

	*j = *jwk

	return nil
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

func (f *base64URLEncodedField) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.RawURLEncoding.EncodeToString(f.data))
}

func (f *base64URLEncodedField) UnmarshalJSON(data []byte) error {
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

	*f = *newBase64URLEncodedField(decoded)

	return nil
}

func (f base64URLEncodedField) toBigInt() *big.Int {
	return new(big.Int).SetBytes(f.data)
}

func (f base64URLEncodedField) toInt() int {
	return int(f.toBigInt().Int64())
}

// Base64 encoded

type base64EncodedField struct {
	data []byte
}

func (f *base64EncodedField) UnmarshalJSON(data []byte) error {
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

	*f = base64EncodedField{
		data: decoded,
	}

	return nil
}

func GetCurve(name string) (elliptic.Curve, error) {
	switch name {
	case "P-256":
		return elliptic.P256(), nil
	case "P-384":
		return elliptic.P384(), nil
	case "P-521":
		return elliptic.P521(), nil
	}
	return nil, fmt.Errorf("invalid crv: %s", name)
}

func getPublicKeyFromX5c(x5c []*base64EncodedField) (interface{}, error) {
	if len(x5c) == 0 {
		return nil, nil
	}

	certificate, err := x509.ParseCertificate(x5c[0].data)
	if err != nil {
		return nil, err
	}
	return certificate.PublicKey, nil
}
