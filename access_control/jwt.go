package access_control

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
)

const (
	AlgorithmUnknown Algorithm = iota - 1
	_
	AlgorithmRSA
	AlgorithmHMAC
)

const (
	Unknown Source = iota - 1
	Cookie
	Header
)

var (
	ErrorBearerRequired = errors.New("authorization header value must start with 'Bearer '")
	ErrorEmptyToken     = errors.New("empty token")
	ErrorMissingKey     = errors.New("either key_file or key must be specified")
	ErrorNotConfigured  = errors.New("jwt handler not configured")
	ErrorNotSupported   = errors.New("only RSA and HMAC key encodings are supported")
	ErrorUnknownSource  = errors.New("unknown source definition")

	_ AccessControl = &JWT{}
)

type (
	Algorithm int
	Claims    struct{ Audience, Issuer string }
	Source    int
)

type JWT struct {
	algorithm  Algorithm
	source     Source
	sourceKey  string
	hmacSecret []byte
	parser     *jwt.Parser
	pubKey     *rsa.PublicKey
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(algorithm string, claims Claims, src Source, srcKey string, key []byte) (*JWT, error) {
	if len(key) == 0 {
		return nil, ErrorMissingKey
	}

	if src == Unknown {
		return nil, ErrorUnknownSource
	}

	algo := newAlgorithm(algorithm)
	if algo == AlgorithmUnknown {
		return nil, ErrorNotSupported
	}

	jwtObj := &JWT{
		algorithm:  algo,
		hmacSecret: key,
		parser:     newParser(algo, claims.Issuer, claims.Audience),
		source:     src,
		sourceKey:  srcKey,
	}

	pubKey, err := parsePublicPEMKey(key)
	if err != nil && (err != jwt.ErrKeyMustBePEMEncoded || err != jwt.ErrNotRSAPublicKey) {
		cert, err := x509.ParseCertificate(key)
		if err != nil {
			decKey, err := base64.StdEncoding.DecodeString(string(key))
			if err != nil {
				return nil, ErrorNotSupported
			}
			cert, err = x509.ParseCertificate(decKey)
			if err != nil {
				return nil, err
			}
		}
		rsaPubKey, _ := cert.PublicKey.(*rsa.PublicKey)
		pubKey = &rsa.PublicKey{N: rsaPubKey.N, E: rsaPubKey.E}
	}
	jwtObj.pubKey = pubKey
	return jwtObj, err
}

// Validate reading the token from configured source and validates against the key.
func (j *JWT) Validate(req *http.Request) error {
	var tokenValue string
	var err error

	if j == nil {
		return ErrorNotConfigured
	}

	switch j.source {
	case Cookie:
		if cookie, err := req.Cookie(j.sourceKey); err != nil && err != http.ErrNoCookie {
			return err
		} else if cookie != nil {
			tokenValue = cookie.Value
		}
	case Header:
		if j.sourceKey == "Authorization" {
			if tokenValue = req.Header.Get(j.sourceKey); tokenValue == "" {
				return ErrorEmptyToken
			}

			if tokenValue, err = getBearer(tokenValue); err != nil {
				return err
			}
		} else {
			tokenValue = req.Header.Get(j.sourceKey)
		}
	}

	// TODO j.PostParam, j.QueryParam
	if tokenValue == "" {
		return ErrorEmptyToken
	}

	_, err = j.parser.Parse(tokenValue, func(_ *jwt.Token) (interface{}, error) {
		switch j.algorithm {
		case AlgorithmRSA:
			return j.pubKey, nil
		case AlgorithmHMAC:
			return j.hmacSecret, nil
		default:
			return nil, ErrorNotSupported
		}
	})
	return err
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", ErrorBearerRequired
}

func newParser(algo Algorithm, iss, aud string) *jwt.Parser {
	var options []jwt.ParserOption
	options = append(options, jwt.WithValidMethods([]string{algo.String()}))
	if iss != "" {
		options = append(options, jwt.WithIssuer(iss))
	}
	if aud != "" {
		options = append(options, jwt.WithAudience(aud))
	} else {
		options = append(options, jwt.WithoutAudienceValidation())
	}
	return jwt.NewParser(options...)
}

func newAlgorithm(a string) Algorithm {
	switch a {
	case "RS256":
		return AlgorithmRSA
	case "HS256":
		return AlgorithmHMAC
	default:
		return AlgorithmUnknown
	}
}

func (a Algorithm) String() string {
	switch a {
	case AlgorithmRSA:
		return "RS256"
	case AlgorithmHMAC:
		return "HS256"
	default:
		return "Unknown"
	}
}

// parsePublicPEMKey tries to parse all supported publicKey variations which
// must be given in PEM encoded format.
func parsePublicPEMKey(key []byte) (pub *rsa.PublicKey, err error) {
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		return nil, jwt.ErrKeyMustBePEMEncoded
	}
	pubKey, pubErr := x509.ParsePKCS1PublicKey(pemBlock.Bytes)
	if pubErr != nil {
		pkixKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
		if err != nil {
			cert, cerr := x509.ParseCertificate(pemBlock.Bytes)
			if cerr != nil {
				return nil, jwt.ErrNotRSAPublicKey
			}
			if k, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				return k, nil
			}
			return nil, jwt.ErrNotRSAPublicKey
		}
		if k, ok := pkixKey.(*rsa.PublicKey); !ok {
			return nil, jwt.ErrNotRSAPublicKey
		} else {
			pubKey = k
		}
	}
	return pubKey, nil
}
