package accesscontrol

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
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
	Claims    map[string]interface{}
	Source    int
)

type JWT struct {
	algorithm      Algorithm
	claims         Claims
	claimsRequired []string
	ignoreExp      bool
	source         Source
	sourceKey      string
	hmacSecret     []byte
	name           string
	parser         *jwt.Parser
	pubKey         *rsa.PublicKey
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(algorithm, name string, claims Claims, reqClaims []string, src Source, srcKey string, key []byte) (*JWT, error) {
	if len(key) == 0 {
		return nil, ErrorMissingKey
	}

	if src == Unknown {
		return nil, ErrorUnknownSource
	}

	algo := NewAlgorithm(algorithm)
	if algo == AlgorithmUnknown {
		return nil, ErrorNotSupported
	}

	parser, err := newParser(algo, claims)
	if err != nil {
		return nil, err
	}

	jwtObj := &JWT{
		algorithm:      algo,
		claims:         claims,
		claimsRequired: reqClaims,
		hmacSecret:     key,
		name:           name,
		parser:         parser,
		source:         src,
		sourceKey:      srcKey,
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

	token, err := j.parser.ParseWithClaims(tokenValue, jwt.MapClaims{}, j.getValidationKey)
	if err != nil {
		return err
	}

	tokenClaims, err := j.validateClaims(token)
	if err != nil {
		return err
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(ContextAccessControlKey).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[j.name] = tokenClaims
	ctx = context.WithValue(ctx, ContextAccessControlKey, acMap)
	*req = *req.WithContext(ctx)

	return nil
}

func (j *JWT) getValidationKey(_ *jwt.Token) (interface{}, error) {
	switch j.algorithm {
	case AlgorithmRSA256, AlgorithmRSA384, AlgorithmRSA512:
		return j.pubKey, nil
	case AlgorithmHMAC256, AlgorithmHMAC384, AlgorithmHMAC512:
		return j.hmacSecret, nil
	default:
		return nil, ErrorNotSupported
	}
}

func (j *JWT) validateClaims(token *jwt.Token) (Claims, error) {
	var tokenClaims jwt.MapClaims
	if tc, ok := token.Claims.(jwt.MapClaims); ok {
		tokenClaims = tc
	}

	if tokenClaims == nil {
		return nil, &jwt.InvalidClaimsError{Message: "token claims has to be a map type"}
	}

	for _, key := range j.claimsRequired {
		if _, ok := tokenClaims[key]; !ok {
			return nil, &jwt.InvalidClaimsError{Message: "required claim is missing: " + key}
		}
	}

	for k, v := range j.claims {

		if k == "iss" || k == "aud" { // gets validated during parsing
			continue
		}

		val, exist := tokenClaims[k]
		if !exist {
			return nil, errors.New("expected claim not found: '" + k + "'")
		}

		if val != v {
			return nil, errors.New("unexpected value for claim '" + k + "'")
		}
	}
	return Claims(tokenClaims), nil
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", ErrorBearerRequired
}

func newParser(algo Algorithm, claims Claims) (*jwt.Parser, error) {
	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{algo.String()}),
		jwt.WithLeeway(time.Second),
	}

	if claims == nil {
		options = append(options, jwt.WithoutAudienceValidation())
		return jwt.NewParser(options...), nil
	}

	if iss, ok := claims["iss"]; ok {
		if err := isStringType(iss); err != nil {
			return nil, fmt.Errorf("iss: %w", err)
		}
		options = append(options, jwt.WithIssuer(iss.(string)))
	}
	if aud, ok := claims["aud"]; ok {
		if err := isStringType(aud); err != nil {
			return nil, fmt.Errorf("aud: %w", err)
		}
		options = append(options, jwt.WithAudience(aud.(string)))
	} else {
		options = append(options, jwt.WithoutAudienceValidation())
	}

	return jwt.NewParser(options...), nil
}

// parsePublicPEMKey tries to parse all supported publicKey variations which
// must be given in PEM encoded format.
func parsePublicPEMKey(key []byte) (pub *rsa.PublicKey, err error) {
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		decKey, err := base64.StdEncoding.DecodeString(string(key))
		if err != nil {
			return nil, ErrorNotSupported
		}
		pemBlock, _ = pem.Decode(decKey)
		if pemBlock == nil {
			return nil, jwt.ErrKeyMustBePEMEncoded
		}
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

func isStringType(val interface{}) error {
	switch val.(type) {
	case string:
		return nil
	default:
		return errors.New("expected a string value type")
	}
}
