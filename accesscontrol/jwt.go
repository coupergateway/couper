package accesscontrol

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

const (
	Invalid JWTSourceType = iota
	Cookie
	Header

	jwtErrorKind = "jwt"
)

var _ AccessControl = &JWT{}

var JWTError = errors.AccessControl.Kind(jwtErrorKind)

type (
	Algorithm     int
	JWTSourceType uint8
	JWTSource     struct {
		Name string
		Type JWTSourceType
	}
)

type JWT struct {
	algorithm      Algorithm
	claims         map[string]interface{}
	claimsRequired []string
	source         JWTSource
	hmacSecret     []byte
	name           string
	parser         *jwt.Parser
	pubKey         *rsa.PublicKey
}

type JWTOptions struct {
	Algorithm      string
	Claims         map[string]interface{}
	ClaimsRequired []string
	Name           string // TODO: more generic (validate)
	Source         JWTSource
	Key            string
	KeyFile        string
}

func NewJWTSource(cookie, header string) JWTSource {
	if cookie != "" && header != "" { // invalid
		return JWTSource{}
	}

	if cookie != "" {
		return JWTSource{
			Name: cookie,
			Type: Cookie,
		}
	} else if header != "" {
		return JWTSource{
			Name: header,
			Type: Header,
		}
	}
	return JWTSource{}
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(options *JWTOptions) (*JWT, error) {
	confErr := errors.Configuration.Label(options.Name)

	jwtAC := &JWT{
		algorithm:      NewAlgorithm(options.Algorithm),
		claims:         options.Claims,
		claimsRequired: options.ClaimsRequired,
		name:           options.Name,
		source:         options.Source,
	}

	if options.Key != "" && options.KeyFile != "" {
		return nil, confErr.Message("key and keyFile provided")
	}

	key := []byte(options.Key)
	if options.KeyFile != "" {
		k, err := readKeyFile(options.KeyFile)
		if err != nil {
			return nil, confErr.With(err)
		}
		key = k
	}

	if len(key) == 0 {
		return nil, confErr.Message("key required")
	}

	if jwtAC.source.Type == Invalid {
		return nil, confErr.Message("token source is invalid")
	}

	if jwtAC.algorithm == AlgorithmUnknown {
		return nil, confErr.Message("algorithm is not supported")
	}

	parser, err := newParser(jwtAC.algorithm, jwtAC.claims)
	if err != nil {
		return nil, confErr.With(err)
	}
	jwtAC.parser = parser

	if jwtAC.algorithm.IsHMAC() {
		jwtAC.hmacSecret = key
		return jwtAC, nil
	}

	pubKey, err := parsePublicPEMKey(key)
	if err != nil {
		return nil, confErr.With(err)
	}

	jwtAC.pubKey = pubKey
	return jwtAC, nil
}

// Validate reading the token from configured source and validates against the key.
func (j *JWT) Validate(req *http.Request) error {
	// TODO: refactor to more generic validateFunc location with Kind interface for kind prefix
	err := j.validate(req)

	if j == nil { // we could not obtain the label
		return err
	}

	switch err.(type) {
	case nil:
		return nil
	case *errors.Error:
		return err.(*errors.Error).Label(j.name)
	default:
		return JWTError.Label(j.name).With(err)
	}
}

func (j *JWT) validate(req *http.Request) error {
	var tokenValue string
	var err error

	if j == nil {
		return errors.Configuration
	}

	switch j.source.Type {
	case Cookie:
		if cookie, cerr := req.Cookie(j.source.Name); cerr != nil && cerr != http.ErrNoCookie {
			return cerr
		} else if cookie != nil {
			tokenValue = cookie.Value
		}
	case Header:
		if j.source.Name == "Authorization" {
			if tokenValue = req.Header.Get(j.source.Name); tokenValue != "" {
				if tokenValue, err = getBearer(tokenValue); err != nil {
					return err
				}
			}
		} else {
			tokenValue = req.Header.Get(j.source.Name)
		}
	}

	// TODO j.PostParam, j.QueryParam
	if tokenValue == "" {
		return JWTError.Message("token required").Status(http.StatusUnauthorized)
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
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[j.name] = tokenClaims

	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}

func (j *JWT) getValidationKey(_ *jwt.Token) (interface{}, error) {
	switch j.algorithm {
	case AlgorithmRSA256, AlgorithmRSA384, AlgorithmRSA512:
		return j.pubKey, nil
	case AlgorithmHMAC256, AlgorithmHMAC384, AlgorithmHMAC512:
		return j.hmacSecret, nil
	default: // this error case gets normally caught on configuration level
		return nil, errors.Configuration.Message("algorithm is not supported")
	}
}

func (j *JWT) validateClaims(token *jwt.Token) (map[string]interface{}, error) {
	var tokenClaims jwt.MapClaims
	if tc, ok := token.Claims.(jwt.MapClaims); ok {
		tokenClaims = tc
	}

	const claimErrKind = jwtErrorKind + "_claims_"
	if tokenClaims == nil {
		return nil, JWTError.Kind(claimErrKind + "invalid").Message("token claims has to be a map type")
	}

	for _, key := range j.claimsRequired {
		if _, ok := tokenClaims[key]; !ok {
			return nil, JWTError.Kind(claimErrKind + "missing").Message("required claim is missing: " + key)
		}
	}

	for k, v := range j.claims {

		if k == "iss" || k == "aud" { // gets validated during parsing
			continue
		}

		val, exist := tokenClaims[k]
		if !exist {
			return nil, JWTError.Kind(claimErrKind + "required").Message("missing claim: " + k)
		}

		if val != v {
			return nil, JWTError.Kind(claimErrKind+"invalid_value").Messagef("invalid claim value: %s", val)
		}
	}
	return tokenClaims, nil
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", fmt.Errorf("bearer required")
}

func newParser(algo Algorithm, claims map[string]interface{}) (*jwt.Parser, error) {
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

func readKeyFile(filePath string) ([]byte, error) {
	if filePath != "" {
		p, err := filepath.Abs(filePath)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadFile(p)
	}
	return nil, nil
}

func isStringType(val interface{}) error {
	switch val.(type) {
	case string:
		return nil
	default:
		return fmt.Errorf("invalid value type")
	}
}
