package accesscontrol

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/accesscontrol/jwk"
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
	Value
)

var (
	_ AccessControl         = &JWT{}
	_ DisablePrivateCaching = &JWT{}
)

type (
	JWTSourceType uint8
	JWTSource     struct {
		Expr hcl.Expression
		Name string
		Type JWTSourceType
	}
)

type JWT struct {
	algorithms            []acjwt.Algorithm
	claims                hcl.Expression
	claimsRequired        []string
	disablePrivateCaching bool
	source                JWTSource
	hmacSecret            []byte
	name                  string
	pubKey                interface{}
	rolesClaim            string
	rolesMap              map[string][]string
	permissionsClaim      string
	permissionsMap        map[string][]string
	jwks                  *jwk.JWKS
}

type JWTOptions struct {
	Algorithm             string
	Claims                hcl.Expression
	ClaimsRequired        []string
	DisablePrivateCaching bool
	Name                  string // TODO: more generic (validate)
	RolesClaim            string
	RolesMap              map[string][]string
	PermissionsClaim      string
	PermissionsMap        map[string][]string
	Source                JWTSource
	Key                   []byte
	JWKS                  *jwk.JWKS
}

func NewJWTSource(cookie, header string, value hcl.Expression) JWTSource {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			if h != "" || c != "" {
				return JWTSource{}
			}

			return JWTSource{
				Name: "",
				Type: Value,
				Expr: value,
			}
		}
	}
	if c != "" && h == "" {
		return JWTSource{
			Name: c,
			Type: Cookie,
		}
	}
	if h != "" && c == "" {
		return JWTSource{
			Name: h,
			Type: Header,
		}
	}
	if h == "" && c == "" {
		return JWTSource{
			Name: "Authorization",
			Type: Header,
		}
	}
	return JWTSource{}
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(options *JWTOptions) (*JWT, error) {
	jwtAC, err := newJWT(options)
	if err != nil {
		return nil, err
	}

	algorithm := acjwt.NewAlgorithm(options.Algorithm)
	if algorithm == acjwt.AlgorithmUnknown {
		return nil, fmt.Errorf("algorithm %q is not supported", options.Algorithm)
	}

	jwtAC.algorithms = []acjwt.Algorithm{algorithm}

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
	jwtAC, err := newJWT(options)
	if err != nil {
		return nil, err
	}

	if options.JWKS == nil {
		return nil, fmt.Errorf("invalid JWKS")
	}

	jwtAC.algorithms = append(acjwt.RSAAlgorithms, acjwt.ECDSAlgorithms...)
	jwtAC.jwks = options.JWKS

	return jwtAC, nil
}

func newJWT(options *JWTOptions) (*JWT, error) {
	if options.Source.Type == Invalid {
		return nil, fmt.Errorf("token source is invalid")
	}

	if options.RolesClaim != "" && options.RolesMap == nil {
		return nil, fmt.Errorf("missing beta_roles_map")
	}

	jwtAC := &JWT{
		claims:                options.Claims,
		claimsRequired:        options.ClaimsRequired,
		disablePrivateCaching: options.DisablePrivateCaching,
		name:                  options.Name,
		rolesClaim:            options.RolesClaim,
		rolesMap:              options.RolesMap,
		permissionsClaim:      options.PermissionsClaim,
		permissionsMap:        options.PermissionsMap,
		source:                options.Source,
	}
	return jwtAC, nil
}

func (j *JWT) DisablePrivateCaching() bool {
	return j.disablePrivateCaching
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
					return errors.JwtTokenMissing.With(err)
				}
			}
		} else {
			tokenValue = req.Header.Get(j.source.Name)
		}
	case Value:
		requestContext := eval.ContextFromRequest(req).HCLContext()
		value, diags := eval.Value(requestContext, j.source.Expr)
		if diags != nil {
			return diags
		}

		tokenValue = seetie.ValueToString(value)
	}

	if tokenValue == "" {
		return errors.JwtTokenMissing.Message("token required")
	}

	claims, iss, aud, err := j.getConfiguredClaims(req)
	if err != nil {
		return err
	}

	ctx := req.Context()
	if j.jwks != nil {
		// load JWKS if needed and associate with request uid
		j.jwks.Data(ctx.Value(request.UID).(string))
	}

	parser := newParser(j.algorithms, iss, aud)
	token, err := parser.Parse(tokenValue, j.getValidationKey)
	if err != nil {
		switch err := err.(type) {
		case *jwt.TokenExpiredError:
			return errors.JwtTokenExpired.With(err)
		case *jwt.UnverfiableTokenError:
			if unwrappedError := err.ErrorWrapper.Unwrap(); unwrappedError != nil {
				return errors.JwtTokenInvalid.With(unwrappedError)
			}
		}
		return errors.JwtTokenInvalid.With(err)
	}

	tokenClaims, err := j.validateClaims(token, claims)
	if err != nil {
		return errors.JwtTokenInvalid.With(err)
	}

	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[j.name] = tokenClaims
	ctx = context.WithValue(ctx, request.AccessControls, acMap)

	log := req.Context().Value(request.LogEntry).(*logrus.Entry).WithContext(req.Context())
	scopesValues := j.getScopeValues(tokenClaims, log)

	scopes, _ := ctx.Value(request.BetaGrantedPermissions).([]string)

	scopes = append(scopes, scopesValues...)

	ctx = context.WithValue(ctx, request.BetaGrantedPermissions, scopes)

	*req = *req.WithContext(ctx)

	return nil
}

func (j *JWT) getValidationKey(token *jwt.Token) (interface{}, error) {
	if j.jwks != nil {
		return j.jwks.GetSigKeyForToken(token)
	}

	switch j.algorithms[0] {
	case acjwt.AlgorithmRSA256, acjwt.AlgorithmRSA384, acjwt.AlgorithmRSA512:
		return j.pubKey, nil
	case acjwt.AlgorithmECDSA256, acjwt.AlgorithmECDSA384, acjwt.AlgorithmECDSA512:
		return j.pubKey, nil
	case acjwt.AlgorithmHMAC256, acjwt.AlgorithmHMAC384, acjwt.AlgorithmHMAC512:
		return j.hmacSecret, nil
	default: // this error case gets normally caught on configuration level
		return nil, errors.Configuration.Message("algorithm is not supported")
	}
}

// getConfiguredClaims evaluates the expected claim values from the configuration, and especially iss and aud
func (j *JWT) getConfiguredClaims(req *http.Request) (map[string]interface{}, string, string, error) {
	claims := make(map[string]interface{})
	var iss, aud string
	if j.claims != nil {
		val, verr := eval.Value(eval.ContextFromRequest(req).HCLContext(), j.claims)
		if verr != nil {
			return nil, "", "", verr
		}
		claims = seetie.ValueToMap(val)

		var ok bool
		if issVal, exists := claims["iss"]; exists {
			iss, ok = issVal.(string)
			if !ok {
				return nil, "", "", errors.Configuration.Message("invalid value type, string expected (claims / iss)")
			}
		}

		if audVal, exists := claims["aud"]; exists {
			aud, ok = audVal.(string)
			if !ok {
				return nil, "", "", errors.Configuration.Message("invalid value type, string expected (claims / aud)")
			}
		}
	}

	return claims, iss, aud, nil
}

// validateClaims validates the token claims against the list of required claims and the expected claims values
func (j *JWT) validateClaims(token *jwt.Token, claims map[string]interface{}) (map[string]interface{}, error) {
	var tokenClaims jwt.MapClaims
	if tc, ok := token.Claims.(jwt.MapClaims); ok {
		tokenClaims = tc
	}

	if tokenClaims == nil {
		return nil, fmt.Errorf("token has no claims")
	}

	for _, key := range j.claimsRequired {
		if _, ok := tokenClaims[key]; !ok {
			return nil, fmt.Errorf("required claim is missing: " + key)
		}
	}

	for k, v := range claims {
		if k == "iss" || k == "aud" { // gets validated during parsing
			continue
		}

		val, exist := tokenClaims[k]
		if !exist {
			return nil, fmt.Errorf("required claim is missing: " + k)
		}

		if val != v {
			return nil, fmt.Errorf("unexpected value for claim %s, got %q, expected %q", k, val, v)
		}
	}
	return tokenClaims, nil
}

func (j *JWT) getScopeValues(tokenClaims map[string]interface{}, log *logrus.Entry) []string {
	var scopeValues []string

	scopeValues = j.addScopeValueFromScope(tokenClaims, scopeValues, log)

	scopeValues = j.addScopeValueFromRoles(tokenClaims, scopeValues, log)

	scopeValues = j.addMappedScopeValues(scopeValues, scopeValues)

	return scopeValues
}

const warnInvalidValueMsg = "invalid %s claim value type, ignoring claim, value %#v"

func (j *JWT) addScopeValueFromScope(tokenClaims map[string]interface{}, scopeValues []string, log *logrus.Entry) []string {
	if j.permissionsClaim == "" {
		return scopeValues
	}

	scopesFromClaim, exists := tokenClaims[j.permissionsClaim]
	if !exists {
		return scopeValues
	}

	// ["foo", "bar"] is stored as []interface{}, not []string, unfortunately
	scopesArray, ok := scopesFromClaim.([]interface{})
	if ok {
		var vals []string
		for _, v := range scopesArray {
			s, ok := v.(string)
			if !ok {
				log.Warn(fmt.Sprintf(warnInvalidValueMsg, "permissions", scopesFromClaim))
				return scopeValues
			}
			vals = append(vals, s)
		}
		for _, val := range vals {
			scopeValues, _ = addScopeValue(scopeValues, val)
		}
	} else {
		scopesString, ok := scopesFromClaim.(string)
		if !ok {
			log.Warn(fmt.Sprintf(warnInvalidValueMsg, "permissions", scopesFromClaim))
			return scopeValues
		}
		for _, s := range strings.Split(scopesString, " ") {
			scopeValues, _ = addScopeValue(scopeValues, s)
		}
	}
	return scopeValues
}

func (j *JWT) getRoleValues(rolesClaimValue interface{}, log *logrus.Entry) []string {
	var roleValues []string
	// ["foo", "bar"] is stored as []interface{}, not []string, unfortunately
	rolesArray, ok := rolesClaimValue.([]interface{})
	if ok {
		var vals []string
		for _, v := range rolesArray {
			r, ok := v.(string)
			if !ok {
				log.Warn(fmt.Sprintf(warnInvalidValueMsg, "roles", rolesClaimValue))
				return roleValues
			}
			vals = append(vals, r)
		}
		return vals
	} else {
		rolesString, ok := rolesClaimValue.(string)
		if !ok {
			log.Warn(fmt.Sprintf(warnInvalidValueMsg, "roles", rolesClaimValue))
			return roleValues
		}
		return strings.Split(rolesString, " ")
	}
}

func (j *JWT) addScopeValueFromRoles(tokenClaims map[string]interface{}, scopeValues []string, log *logrus.Entry) []string {
	if j.rolesClaim == "" || j.rolesMap == nil {
		return scopeValues
	}

	rolesClaimValue, exists := tokenClaims[j.rolesClaim]
	if !exists {
		return scopeValues
	}

	roleValues := j.getRoleValues(rolesClaimValue, log)
	for _, r := range roleValues {
		if scopes, exist := j.rolesMap[r]; exist {
			for _, s := range scopes {
				scopeValues, _ = addScopeValue(scopeValues, s)
			}
		}
	}

	if scopes, exist := j.rolesMap["*"]; exist {
		for _, s := range scopes {
			scopeValues, _ = addScopeValue(scopeValues, s)
		}
	}
	return scopeValues
}

func (j *JWT) addMappedScopeValues(source, target []string) []string {
	if j.permissionsMap == nil {
		return target
	}

	for _, val := range source {
		mappedValues, exist := j.permissionsMap[val]
		if !exist {
			// no mapping for value
			continue
		}

		var l []string
		for _, mv := range mappedValues {
			var added bool
			// add value from mapping?
			target, added = addScopeValue(target, mv)
			if !added {
				continue
			}
			l = append(l, mv)
		}
		// recursion: call only with values not already in target
		target = j.addMappedScopeValues(l, target)
	}
	return target
}

func addScopeValue(scopeValues []string, scope string) ([]string, bool) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return scopeValues, false
	}
	for _, s := range scopeValues {
		if s == scope {
			return scopeValues, false
		}
	}
	return append(scopeValues, scope), true
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", fmt.Errorf("bearer required with authorization header")
}

// newParser creates a new parser with issuer/audience validation if configured via iss/aud in expected claims
func newParser(algos []acjwt.Algorithm, iss, aud string) *jwt.Parser {
	var algorithms []string
	for _, a := range algos {
		algorithms = append(algorithms, a.String())
	}
	options := []jwt.ParserOption{
		jwt.WithValidMethods(algorithms),
		jwt.WithLeeway(time.Second),
	}

	if aud != "" {
		options = append(options, jwt.WithAudience(aud))
	} else {
		options = append(options, jwt.WithoutAudienceValidation())
	}
	if iss != "" {
		options = append(options, jwt.WithIssuer(iss))
	}

	return jwt.NewParser(options...)
}

// parsePublicPEMKey tries to parse all supported publicKey variations which
// must be given in PEM encoded format.
func parsePublicPEMKey(key []byte) (pub interface{}, err error) {
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
			if k, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
				return k, nil
			}

			return nil, fmt.Errorf("invalid RSA/ECDSA public key")
		}

		if k, ok := pkixKey.(*rsa.PublicKey); ok {
			return k, nil
		}

		if k, ok := pkixKey.(*ecdsa.PublicKey); ok {
			return k, nil
		}

		return nil, fmt.Errorf("invalid RSA/ECDSA public key")
	}
	return pubKey, nil
}
