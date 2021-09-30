package accesscontrol

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"

	acjwt "github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/seetie"
)

const (
	Invalid JWTSourceType = iota
	Cookie
	Header
)

var _ AccessControl = &JWT{}

type (
	JWTSourceType uint8
	JWTSource     struct {
		Name string
		Type JWTSourceType
	}
)

type JWT struct {
	algorithms     []acjwt.Algorithm
	claims         hcl.Expression
	claimsRequired []string
	source         JWTSource
	hmacSecret     []byte
	name           string
	pubKey         *rsa.PublicKey
	roleClaim      string
	roleMap        map[string][]string
	scopeClaim     string
	jwks           *JWKS
}

type JWTOptions struct {
	Algorithm      string
	Claims         hcl.Expression
	ClaimsRequired []string
	Name           string // TODO: more generic (validate)
	RoleClaim      string
	RoleMap        map[string][]string
	ScopeClaim     string
	Source         JWTSource
	Key            []byte
	JWKS           *JWKS
}

func NewJWTSource(cookie, header string) JWTSource {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)
	if c != "" && h != "" { // both are invalid
		return JWTSource{}
	}

	if c != "" {
		return JWTSource{
			Name: c,
			Type: Cookie,
		}
	} else if h != "" {
		return JWTSource{
			Name: h,
			Type: Header,
		}
	}
	return JWTSource{}
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(options *JWTOptions) (*JWT, error) {
	err := checkOptions(options)
	if err != nil {
		return nil, err
	}

	algorithm := acjwt.NewAlgorithm(options.Algorithm)
	if algorithm == acjwt.AlgorithmUnknown {
		return nil, fmt.Errorf("algorithm %q is not supported", options.Algorithm)
	}

	jwtAC := &JWT{
		algorithms:     []acjwt.Algorithm{algorithm},
		claims:         options.Claims,
		claimsRequired: options.ClaimsRequired,
		name:           options.Name,
		roleClaim:      options.RoleClaim,
		roleMap:        options.RoleMap,
		scopeClaim:     options.ScopeClaim,
		source:         options.Source,
	}

	if algorithm.IsHMAC() {
		jwtAC.hmacSecret = options.Key
		return jwtAC, nil
	}

	pubKey, err := parsePublicPEMKey(options.Key)
	if err != nil {
		return nil, err
	}

	jwtAC.pubKey = pubKey
	return jwtAC, nil
}

func NewJWTFromJWKS(options *JWTOptions) (*JWT, error) {
	err := checkOptions(options)
	if err != nil {
		return nil, err
	}

	jwtAC := &JWT{
		algorithms:     acjwt.RSAAlgorithms,
		claims:         options.Claims,
		claimsRequired: options.ClaimsRequired,
		name:           options.Name,
		roleClaim:      options.RoleClaim,
		roleMap:        options.RoleMap,
		scopeClaim:     options.ScopeClaim,
		source:         options.Source,
		jwks:           options.JWKS,
	}

	if jwtAC.jwks == nil {
		return nil, fmt.Errorf("invalid JWKS")
	}

	return jwtAC, nil
}

func checkOptions(options *JWTOptions) error {
	if options.Source.Type == Invalid {
		return fmt.Errorf("token source is invalid")
	}

	if options.RoleClaim != "" && options.RoleMap == nil {
		return fmt.Errorf("missing beta_role_map")
	}

	return nil
}

// Validate reading the token from configured source and validates against the key.
func (j *JWT) Validate(req *http.Request) error {
	var tokenValue string
	var err error

	switch j.source.Type {
	case Cookie:
		cookie, cerr := req.Cookie(j.source.Name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case Header:
		if strings.ToLower(j.source.Name) == "authorization" {
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
		return errors.JwtTokenMissing.Message("token required")
	}

	claims := make(map[string]interface{})
	var diags hcl.Diagnostics
	if j.claims != nil {
		claims, diags = seetie.ExpToMap(eval.ContextFromRequest(req).HCLContext(), j.claims)
		if diags != nil {
			return diags
		}
	}

	parser, err := newParser(j.algorithms, claims)
	if err != nil {
		return err
	}

	token, err := parser.Parse(tokenValue, j.getValidationKey)
	if err != nil {
		switch err.(type) {
		case *jwt.TokenExpiredError:
			return errors.JwtTokenExpired.With(err)
		case *jwt.UnverfiableTokenError:
			return err.(*jwt.UnverfiableTokenError).ErrorWrapper.Unwrap()
		default:
			return err
		}
	}

	tokenClaims, err := j.validateClaims(token, claims)
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

	scopesValues, err := j.getScopeValues(tokenClaims)
	if err != nil {
		return err
	}

	if len(scopesValues) > 0 {
		scopes, ok := ctx.Value(request.Scopes).([]string)
		if !ok {
			scopes = []string{}
		}
		for _, sc := range scopesValues {
			scopes = append(scopes, sc)
		}
		ctx = context.WithValue(ctx, request.Scopes, scopes)
	}

	*req = *req.WithContext(ctx)

	return nil
}

func (j *JWT) getValidationKey(token *jwt.Token) (interface{}, error) {
	if j.jwks != nil {
		id := token.Header["kid"]
		algorithm := token.Header["alg"]
		if id == nil {
			return nil, fmt.Errorf("Missing \"kid\" in JOSE header")
		}
		if algorithm == nil {
			return nil, fmt.Errorf("Missing \"alg\" in JOSE header")
		}
		jwk, err := j.jwks.GetKey(id.(string), algorithm.(string), "sig")
		if err != nil {
			return nil, err
		}

		if jwk == nil {
			return nil, fmt.Errorf("No matching %s JWK for kid %q", algorithm, id)
		}

		return jwk.Key, nil
	}

	switch j.algorithms[0] {
	case acjwt.AlgorithmRSA256, acjwt.AlgorithmRSA384, acjwt.AlgorithmRSA512:
		return j.pubKey, nil
	case acjwt.AlgorithmHMAC256, acjwt.AlgorithmHMAC384, acjwt.AlgorithmHMAC512:
		return j.hmacSecret, nil
	default: // this error case gets normally caught on configuration level
		return nil, errors.Configuration.Message("algorithm is not supported")
	}
}

func (j *JWT) validateClaims(token *jwt.Token, claims map[string]interface{}) (map[string]interface{}, error) {
	var tokenClaims jwt.MapClaims
	if tc, ok := token.Claims.(jwt.MapClaims); ok {
		tokenClaims = tc
	}

	if tokenClaims == nil {
		return nil, errors.JwtTokenInvalid.Message("token has no claims")
	}

	for _, key := range j.claimsRequired {
		if _, ok := tokenClaims[key]; !ok {
			return nil, errors.JwtTokenInvalid.Message("required claim is missing: " + key)
		}
	}

	for k, v := range claims {

		if k == "iss" || k == "aud" { // gets validated during parsing
			continue
		}

		val, exist := tokenClaims[k]
		if !exist {
			return nil, errors.JwtTokenInvalid.Message("required claim is missing: " + k)
		}

		if val != v {
			return nil, errors.JwtTokenInvalid.Messagef("unexpected value for claim %s: %q, expected %q", k, val, v)
		}
	}
	return tokenClaims, nil
}

func (j *JWT) getScopeValues(tokenClaims map[string]interface{}) ([]string, error) {
	scopeValues := []string{}

	if j.scopeClaim != "" {
		scopesFromClaim, exists := tokenClaims[j.scopeClaim]
		if !exists {
			return nil, fmt.Errorf("Missing expected scope claim %q", j.scopeClaim)
		}

		// ["foo", "bar"] is stored as []interface{}, not []string, unfortunately
		scopesArray, ok := scopesFromClaim.([]interface{})
		if ok {
			for _, v := range scopesArray {
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("value of scope claim must either be a string containing a space-separated list of scope values or a list of string scope values")
				}
				scopeValues = addScopeValue(scopeValues, s)
			}
		} else {
			scopesString, ok := scopesFromClaim.(string)
			if !ok {
				return nil, fmt.Errorf("value of scope claim must either be a string containing a space-separated list of scope values or a list of string scope values")
			}
			for _, s := range strings.Split(scopesString, " ") {
				scopeValues = addScopeValue(scopeValues, s)
			}
		}
	}

	if j.roleClaim != "" {
		rolesClaimValue, exists := tokenClaims[j.roleClaim]
		if !exists {
			return nil, fmt.Errorf("Missing expected role claim %q", j.roleClaim)
		}

		roleValues := []string{}
		// ["foo", "bar"] is stored as []interface{}, not []string, unfortunately
		rolesArray, ok := rolesClaimValue.([]interface{})
		if ok {
			for _, v := range rolesArray {
				r, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("value of role claim must either be a string containing a space-separated list of scope values or a list of string scope values")
				}
				roleValues = append(roleValues, r)
			}
		} else {
			rolesString, ok := rolesClaimValue.(string)
			if !ok {
				return nil, fmt.Errorf("value of role claim must either be a string containing a space-separated list of scope values or a list of string scope values")
			}
			roleValues = strings.Split(rolesString, " ")
		}
		for _, r := range roleValues {
			if scopes, exists := j.roleMap[r]; exists {
				for _, s := range scopes {
					scopeValues = addScopeValue(scopeValues, s)
				}
			}
		}
	}

	return scopeValues, nil
}

func addScopeValue(scopeValues []string, scope string) []string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return scopeValues
	}
	for _, s := range scopeValues {
		if s == scope {
			return scopeValues
		}
	}
	return append(scopeValues, scope)
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", errors.JwtTokenExpired.Message("bearer required with authorization header")
}

func newParser(algos []acjwt.Algorithm, claims map[string]interface{}) (*jwt.Parser, error) {
	var algorithms []string
	for _, a := range algos {
		algorithms = append(algorithms, a.String())
	}
	options := []jwt.ParserOption{
		jwt.WithValidMethods(algorithms),
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
		pkixKey, pkerr := x509.ParsePKIXPublicKey(pemBlock.Bytes)
		if pkerr != nil {
			cert, cerr := x509.ParseCertificate(pemBlock.Bytes)
			if cerr != nil {
				return nil, jwt.ErrNotRSAPublicKey
			}
			if k, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				return k, nil
			}
			return nil, jwt.ErrNotRSAPublicKey
		}
		k, ok := pkixKey.(*rsa.PublicKey)
		if !ok {
			return nil, jwt.ErrNotRSAPublicKey
		}
		pubKey = k
	}
	return pubKey, nil
}

func isStringType(val interface{}) error {
	switch val.(type) {
	case string:
		return nil
	default:
		return fmt.Errorf("invalid value type")
	}
}
