package accesscontrol

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	goerrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/accesscontrol/jwk"
	acjwt "github.com/coupergateway/couper/accesscontrol/jwt"
	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

var (
	_ AccessControl         = &JWT{}
	_ DisablePrivateCaching = &JWT{}
)

type JWT struct {
	algos                 []string
	claims                hcl.Expression
	claimsRequired        []string
	disablePrivateCaching bool
	source                *TokenSource
	hmacSecret            []byte
	name                  string
	pubKey                interface{}
	rolesClaim            string
	rolesMap              map[string][]string
	permissionsClaim      string
	permissionsMap        map[string][]string
	jwks                  *jwk.JWKS
	memStore              *cache.MemoryStore
}

// NewJWT parses the key and creates Validation obj which can be referenced in related handlers.
func NewJWT(jwtConf *config.JWT, key []byte, memStore *cache.MemoryStore) (*JWT, error) {
	jwtAC, err := newJWT(jwtConf, memStore)
	if err != nil {
		return nil, err
	}

	algorithm := acjwt.NewAlgorithm(jwtConf.SignatureAlgorithm)
	if algorithm == acjwt.AlgorithmUnknown {
		return nil, fmt.Errorf("algorithm %q is not supported", jwtConf.SignatureAlgorithm)
	}

	jwtAC.algos = []string{jwtConf.SignatureAlgorithm}

	if algorithm.IsHMAC() {
		jwtAC.hmacSecret = key
		return jwtAC, nil
	}

	pubKey, err := parsePublicPEMKey(key)
	if err != nil {
		return nil, err
	}

	jwtAC.pubKey = pubKey
	return jwtAC, nil
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

func NewJWTFromJWKS(jwtConf *config.JWT, jwks *jwk.JWKS, memStore *cache.MemoryStore) (*JWT, error) {
	if jwks == nil {
		return nil, fmt.Errorf("invalid JWKS")
	}

	jwtAC, err := newJWT(jwtConf, memStore)
	if err != nil {
		return nil, err
	}

	algorithms := append(acjwt.RSAAlgorithms, acjwt.ECDSAlgorithms...)
	var algos []string
	for _, a := range algorithms {
		algos = append(algos, a.String())
	}

	jwtAC.algos = algos
	jwtAC.jwks = jwks

	return jwtAC, nil
}

type parserConfig struct {
	algorithms []string
	audience   string
	issuer     string
}

func (p parserConfig) key() string {
	return fmt.Sprintf("pc:%s:%s:%s", p.algorithms, p.audience, p.issuer)
}

func (p parserConfig) newParser() *jwt.Parser {
	options := []jwt.ParserOption{
		jwt.WithValidMethods(p.algorithms),
		jwt.WithLeeway(time.Second),
	}
	if p.audience != "" {
		options = append(options, jwt.WithAudience(p.audience))
	}
	if p.issuer != "" {
		options = append(options, jwt.WithIssuer(p.issuer))
	}

	return jwt.NewParser(options...)
}

func newJWT(jwtConf *config.JWT, memStore *cache.MemoryStore) (*JWT, error) {
	source, err := NewTokenSource(jwtConf.Bearer, jwtConf.Cookie, jwtConf.Header, jwtConf.TokenValue)
	if err != nil {
		return nil, err
	}

	if jwtConf.RolesClaim != "" && jwtConf.RolesMap == nil {
		return nil, fmt.Errorf("missing roles_map")
	}

	jwtAC := &JWT{
		claims:                jwtConf.Claims,
		claimsRequired:        jwtConf.ClaimsRequired,
		disablePrivateCaching: jwtConf.DisablePrivateCaching,
		memStore:              memStore,
		name:                  jwtConf.Name,
		rolesClaim:            jwtConf.RolesClaim,
		rolesMap:              jwtConf.RolesMap,
		permissionsClaim:      jwtConf.PermissionsClaim,
		permissionsMap:        jwtConf.PermissionsMap,
		source:                source,
	}
	return jwtAC, nil
}

func (j *JWT) DisablePrivateCaching() bool {
	return j.disablePrivateCaching
}

// getParser returns a JWT parser for a parser config
func (j *JWT) getParser(p parserConfig) *jwt.Parser {
	key := p.key()
	if parser, ok := j.memStore.Get(key).(*jwt.Parser); ok {
		return parser
	}

	parser := p.newParser()
	j.memStore.Set(key, parser, 3600)
	return parser
}

// Validate reading the token from configured source and validates against the key.
func (j *JWT) Validate(req *http.Request) error {
	tokenValue, err := j.source.TokenValue(req)
	if err != nil {
		return errors.JwtTokenMissing.With(err)
	}

	expectedClaims, err := j.getConfiguredClaims(req)
	if err != nil {
		return err
	}

	parserConfig := parserConfig{
		algorithms: j.algos,
	}
	if aud, ok := expectedClaims["aud"].(string); ok {
		parserConfig.audience = aud
	}
	if iss, ok := expectedClaims["iss"].(string); ok {
		parserConfig.issuer = iss
	}
	parser := j.getParser(parserConfig)

	if j.jwks != nil {
		// load JWKS if needed
		j.jwks.Data()
	}

	tokenClaims := jwt.MapClaims{}
	_, err = parser.ParseWithClaims(tokenValue, tokenClaims, j.getValidationKey)
	if err != nil {
		if goerrors.Is(err, jwt.ErrTokenExpired) {
			return errors.JwtTokenExpired.With(err)
		}
		if goerrors.Is(err, jwt.ErrTokenInvalidClaims) {
			// TODO throw different error?
			return errors.JwtTokenInvalid.With(err)
		}
		return errors.JwtTokenInvalid.With(err)
	}

	err = j.validateClaims(tokenClaims, expectedClaims)
	if err != nil {
		// TODO throw different error?
		return errors.JwtTokenInvalid.With(err)
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	// treat token claims as map for context
	acMap[j.name] = map[string]interface{}(tokenClaims)
	ctx = context.WithValue(ctx, request.AccessControls, acMap)

	log := req.Context().Value(request.LogEntry).(*logrus.Entry).WithContext(req.Context())
	grantedPermissions := j.getGrantedPermissions(tokenClaims, log)

	alreadyGrantedPermissions, _ := ctx.Value(request.GrantedPermissions).([]string)

	grantedPermissions = append(alreadyGrantedPermissions, grantedPermissions...)

	ctx = context.WithValue(ctx, request.GrantedPermissions, grantedPermissions)

	*req = *req.WithContext(ctx)

	return nil
}

func (j *JWT) getValidationKey(token *jwt.Token) (interface{}, error) {
	if j.jwks != nil {
		return j.jwks.GetSigKeyForToken(token)
	}

	if j.pubKey != nil {
		return j.pubKey, nil
	} else if j.hmacSecret != nil {
		return j.hmacSecret, nil
	} else {
		// this error case gets normally caught on configuration level
		return nil, errors.Configuration.Message("algorithm is not supported")
	}
}

// getConfiguredClaims evaluates the expected claim values from the configuration, and especially iss and aud
func (j *JWT) getConfiguredClaims(req *http.Request) (map[string]interface{}, error) {
	claims := make(map[string]interface{})
	if j.claims == nil { // tests only
		return claims, nil
	}

	val, verr := eval.Value(eval.ContextFromRequest(req).HCLContext(), j.claims)
	if verr != nil {
		return nil, verr
	}

	if val.IsNull() { // claims not configured
		return claims, nil
	}

	claims = seetie.ValueToMap(val)

	var ok bool
	if issVal, exists := claims["iss"]; exists {
		_, ok = issVal.(string)
		if !ok {
			return nil, errors.Configuration.Message("invalid value type, string expected (claims / iss)")
		}
	}

	if audVal, exists := claims["aud"]; exists {
		_, ok = audVal.(string)
		if !ok {
			return nil, errors.Configuration.Message("invalid value type, string expected (claims / aud)")
		}
	}

	return claims, nil
}

// validateClaims validates the token claims against the list of required claims and the expected claims values
func (j *JWT) validateClaims(tokenClaims jwt.MapClaims, expectedClaims map[string]interface{}) error {
	for _, key := range j.claimsRequired {
		if _, ok := tokenClaims[key]; !ok {
			return fmt.Errorf("required claim is missing: " + key)
		}
	}

	for k, v := range expectedClaims {
		val, exist := tokenClaims[k]
		if !exist {
			return fmt.Errorf("required claim is missing: " + k)
		}

		if k == "iss" {
			// ignore, already handled during parsing
			continue
		}
		if k == "aud" {
			// ignore, already handled during parsing
			continue
		}

		if val != v {
			return fmt.Errorf("unexpected value for claim %s, got %q, expected %q", k, val, v)
		}
	}
	return nil
}

func (j *JWT) getGrantedPermissions(tokenClaims jwt.MapClaims, log *logrus.Entry) []string {
	var grantedPermissions []string

	grantedPermissions = j.addPermissionsFromPermissionsClaim(tokenClaims, grantedPermissions, log)

	grantedPermissions = j.addPermissionsFromRoles(tokenClaims, grantedPermissions, log)

	grantedPermissions = j.addMappedPermissions(grantedPermissions, grantedPermissions)

	return grantedPermissions
}

const warnInvalidValueMsg = "invalid %s claim value type, ignoring claim, value %#v"

func (j *JWT) addPermissionsFromPermissionsClaim(tokenClaims jwt.MapClaims, permissions []string, log *logrus.Entry) []string {
	if j.permissionsClaim == "" {
		return permissions
	}

	permissionsFromClaim, exists := tokenClaims[j.permissionsClaim]
	if !exists {
		return permissions
	}

	// ["foo", "bar"] is stored as []interface{}, not []string, unfortunately
	permissionsArray, ok := permissionsFromClaim.([]interface{})
	if ok {
		var vals []string
		for _, v := range permissionsArray {
			p, ok := v.(string)
			if !ok {
				log.Warn(fmt.Sprintf(warnInvalidValueMsg, "permissions", permissionsFromClaim))
				return permissions
			}
			vals = append(vals, p)
		}
		for _, val := range vals {
			permissions, _ = addPermission(permissions, val)
		}
	} else {
		permissionsString, ok := permissionsFromClaim.(string)
		if !ok {
			log.Warn(fmt.Sprintf(warnInvalidValueMsg, "permissions", permissionsFromClaim))
			return permissions
		}
		for _, p := range strings.Split(permissionsString, " ") {
			permissions, _ = addPermission(permissions, p)
		}
	}
	return permissions
}

func getRoleValues(rolesClaimValue interface{}, log *logrus.Entry) []string {
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
	}

	rolesString, ok := rolesClaimValue.(string)
	if !ok {
		log.Warn(fmt.Sprintf(warnInvalidValueMsg, "roles", rolesClaimValue))
		return roleValues
	}
	return strings.Split(rolesString, " ")
}

func (j *JWT) addPermissionsFromRoles(tokenClaims jwt.MapClaims, permissions []string, log *logrus.Entry) []string {
	if j.rolesClaim == "" || j.rolesMap == nil {
		return permissions
	}

	rolesClaimValue, exists := tokenClaims[j.rolesClaim]
	if !exists {
		return permissions
	}

	roleValues := getRoleValues(rolesClaimValue, log)
	for _, r := range roleValues {
		if perms, exist := j.rolesMap[r]; exist {
			for _, p := range perms {
				permissions, _ = addPermission(permissions, p)
			}
		}
	}

	if perms, exist := j.rolesMap["*"]; exist {
		for _, p := range perms {
			permissions, _ = addPermission(permissions, p)
		}
	}
	return permissions
}

func (j *JWT) addMappedPermissions(source, target []string) []string {
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
			target, added = addPermission(target, mv)
			if !added {
				continue
			}
			l = append(l, mv)
		}
		// recursion: call only with values not already in target
		target = j.addMappedPermissions(l, target)
	}
	return target
}

func addPermission(permissions []string, permission string) ([]string, bool) {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return permissions, false
	}
	for _, p := range permissions {
		if p == permission {
			return permissions, false
		}
	}
	return append(permissions, permission), true
}
