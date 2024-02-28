package accesscontrol

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/bits"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/hcl/v2"

	acjwk "github.com/coupergateway/couper/accesscontrol/jwk"
	acjwt "github.com/coupergateway/couper/accesscontrol/jwt"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

const (
	DpopTyp                    = "dpop+jwt"
	bearerType tokenSourceType = iota
	cookieType
	dpopType
	headerType
	valueType
)

type (
	tokenSourceType uint8
)

// TokenSource represents the source from which a token is retrieved.
type TokenSource interface {
	// TokenValue retrieves the token value from the request.
	TokenValue(req *http.Request) (string, error)
	// ValidateTokenClaims validates the token (claims) according to e.g. a specific request header field.
	ValidateTokenClaims(token string, tokenClaims map[string]interface{}, req *http.Request) error
}

// NewTokenSource creates a new token source according to various configuration attributes.
func NewTokenSource(bearer, dpop bool, cookie, header string, value hcl.Expression) (TokenSource, error) {
	c, h := strings.TrimSpace(cookie), strings.TrimSpace(header)

	var b uint8
	t := bearerType // default

	if bearer {
		b |= (1 << bearerType)
	}
	if dpop {
		b |= (1 << dpopType)
		t = dpopType
	}
	if c != "" {
		b |= (1 << cookieType)
		t = cookieType
	}
	if h != "" {
		b |= (1 << headerType)
		t = headerType
	}
	if value != nil {
		v, _ := value.Value(nil)
		if !v.IsNull() {
			b |= (1 << valueType)
			t = valueType
		}
	}
	if bits.OnesCount8(b) > 1 {
		return nil, fmt.Errorf("only one of bearer, cookie, header or token_value attributes is allowed")
	}

	if t == dpopType {
		return newDPoPTokenSource(), nil
	}
	if t == valueType {
		return &ValueTokenSource{
			expr: value,
		}, nil
	}
	ts := &NameTokenSource{
		tsType: t,
	}
	switch t {
	case cookieType:
		ts.name = c
	case headerType:
		ts.name = h
	}
	return ts, nil
}

type NameTokenSource struct {
	name   string
	tsType tokenSourceType
}

func (s *NameTokenSource) TokenValue(req *http.Request) (string, error) {
	var tokenValue string
	var err error

	switch s.tsType {
	case bearerType:
		tokenValue, err = getTokenFromAuthorization(req.Header, "Bearer")
	case cookieType:
		cookie, cerr := req.Cookie(s.name)
		if cerr != http.ErrNoCookie && cookie != nil {
			tokenValue = cookie.Value
		}
	case headerType:
		if strings.ToLower(s.name) == "authorization" {
			tokenValue, err = getTokenFromAuthorization(req.Header, "Bearer")
		} else {
			tokenValue = req.Header.Get(s.name)
		}
	}

	if err != nil {
		return "", err
	}

	if tokenValue == "" {
		return "", fmt.Errorf("token required")
	}

	return tokenValue, nil
}

func (s *NameTokenSource) ValidateTokenClaims(token string, tokenClaims map[string]interface{}, req *http.Request) error {
	return nil
}

// getTokenFromAuthorization retrieves a token for the given auth scheme from the Authorization request header field.
func getTokenFromAuthorization(reqHeaders http.Header, authScheme string) (string, error) {
	authorization := reqHeaders.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	pfx := strings.ToLower(authScheme) + " "
	if strings.HasPrefix(strings.ToLower(authorization), pfx) {
		return strings.Trim(authorization[len(pfx):], " "), nil
	}

	return "", fmt.Errorf("auth scheme %q required in authorization header", authScheme)
}

type ValueTokenSource struct {
	expr hcl.Expression
}

func (s *ValueTokenSource) TokenValue(req *http.Request) (string, error) {
	requestContext := eval.ContextFromRequest(req).HCLContext()
	value, err := eval.Value(requestContext, s.expr)
	if err != nil {
		return "", err
	}

	tokenValue := seetie.ValueToString(value)
	if tokenValue == "" {
		return "", fmt.Errorf("token required")
	}

	return tokenValue, nil
}

func (s *ValueTokenSource) ValidateTokenClaims(token string, tokenClaims map[string]interface{}, req *http.Request) error {
	return nil
}

type DPoPTokenSource struct {
	parser *jwt.Parser
}

func newDPoPTokenSource() TokenSource {
	// 5. the alg JOSE header parameter indicates a registered asymmetric
	//    digital signature algorithm [IANA.JOSE.ALGS], is not none, is
	//    supported by the application, and is acceptable per local policy
	algorithms := append(acjwt.RSAAlgorithms, acjwt.ECDSAlgorithms...)
	var algos []string
	for _, a := range algorithms {
		algos = append(algos, a.String())
	}
	parserConfig := parserConfig{
		algorithms: algos,
	}
	return &DPoPTokenSource{
		parser: parserConfig.newParser(),
	}
}

func (s *DPoPTokenSource) TokenValue(req *http.Request) (string, error) {
	tokenValue, err := getTokenFromAuthorization(req.Header, "DPoP")
	if err != nil {
		return "", err
	}

	if tokenValue == "" {
		return "", fmt.Errorf("token required")
	}

	return tokenValue, nil
}

func (s *DPoPTokenSource) ValidateTokenClaims(token string, tokenClaims map[string]interface{}, req *http.Request) error {
	// checks according to 4.3 Checking DPoP Proofs
	// https://www.rfc-editor.org/rfc/rfc9449.html#name-checking-dpop-proofs
	proof, err := s.getValidatedProof(req, token)
	if err != nil {
		return err
	}

	// type already checked in getJwkAndPubKey()
	jwk, _ := proof.Header["jwk"].(map[string]interface{})
	if err = validateCnfClaim(tokenClaims, jwk); err != nil {
		return err
	}

	return nil
}

func (s *DPoPTokenSource) getValidatedProof(req *http.Request, token string) (*jwt.Token, error) {
	dpop, err := getDPoPValue(req.Header)
	if err != nil {
		return nil, err
	}

	proof, err := s.validateDPoPValue(dpop, token, req)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func getDPoPValue(header http.Header) (string, error) {
	dpopCount := len(header.Values("DPoP"))
	if dpopCount == 0 {
		return "", fmt.Errorf("missing DPoP request header field")
	}

	// 1. There is not more than one DPoP HTTP request header field.
	if dpopCount > 1 {
		return "", fmt.Errorf("too many DPoP request header fields")
	}
	dpop := header.Get("DPoP")
	if dpop == "" {
		return "", fmt.Errorf("empty DPoP proof")
	}

	return dpop, nil
}

func (s *DPoPTokenSource) validateDPoPValue(dpop, token string, req *http.Request) (*jwt.Token, error) {
	proofClaims := jwt.MapClaims{}
	// 2. the DPoP HTTP request header field value is a single well-formed
	//    JWT
	proof, err := s.parser.ParseWithClaims(dpop, proofClaims, getJwkAndPubKey)
	// 2. The DPoP HTTP request header field value is a single and well-formed JWT.
	if err != nil {
		return nil, fmt.Errorf("DPoP proof parse error: " + err.Error())
	}

	// 3. All required claims per Section 4.2 are contained in the JWT.
	if err = validateProofHeader(proof.Header); err != nil {
		return nil, err
	}

	if err = validateProofClaims(proofClaims, req, token); err != nil {
		return nil, err
	}

	return proof, nil
}

func getJwkAndPubKey(proof *jwt.Token) (interface{}, error) {
	// 6. The JWT signature verifies with the public key contained in the
	//    jwk JOSE Header Parameter.
	jwk, ok := proof.Header["jwk"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing jwk JOSE header parameter or wrong type")
	}

	// 7. The jwk JOSE Header Parameter does not contain a private key.
	kty, ok := jwk["kty"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing kty property or wrong type")
	}

	var (
		pubKey interface{}
		err    error
	)
	switch kty {
	case "RSA":
		pubKey, err = getRSAPubKey(jwk)
	case "ECDSA":
		pubKey, err = getECDSAPubKey(jwk)
	default:
		// unsupported algorithms are already handled by JWT parser
	}
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

func getRSAPubKey(jwk map[string]interface{}) (*rsa.PublicKey, error) {
	n, err := getN(jwk)
	if err != nil {
		return nil, err
	}

	e, err := getE(jwk)
	if err != nil {
		return nil, err
	}

	// remove non-required members
	for k := range jwk {
		switch k {
		case "kty", "n", "e":
		default:
			delete(jwk, k)
		}
	}

	return &rsa.PublicKey{
		N: toBigInt(n),
		E: toInt(e),
	}, nil
}

func getN(jwk map[string]interface{}) ([]byte, error) {
	n, ok := jwk["n"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing n property or wrong type")
	}

	return base64.RawURLEncoding.DecodeString(n)
}

func getE(jwk map[string]interface{}) ([]byte, error) {
	e, ok := jwk["e"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing e property or wrong type")
	}

	return base64.RawURLEncoding.DecodeString(e)
}

func getECDSAPubKey(jwk map[string]interface{}) (*ecdsa.PublicKey, error) {
	curve, err := getCurve(jwk)
	if err != nil {
		return nil, err
	}

	x, err := getX(jwk)
	if err != nil {
		return nil, err
	}

	y, err := getY(jwk)
	if err != nil {
		return nil, err
	}

	// remove non-required members
	for k := range jwk {
		switch k {
		case "kty", "crv", "x", "y":
		default:
			delete(jwk, k)
		}
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     toBigInt(x),
		Y:     toBigInt(y),
	}, nil
}

func getCurve(jwk map[string]interface{}) (elliptic.Curve, error) {
	crv, ok := jwk["crv"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing crv property or wrong type")
	}

	return acjwk.GetCurve(crv)
}

func getX(jwk map[string]interface{}) ([]byte, error) {
	x, ok := jwk["x"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing x property or wrong type")
	}

	return base64.RawURLEncoding.DecodeString(x)
}

func getY(jwk map[string]interface{}) ([]byte, error) {
	y, ok := jwk["y"].(string)
	if !ok {
		return nil, fmt.Errorf("jwk JOSE header parameter missing y property or wrong type")
	}

	return base64.RawURLEncoding.DecodeString(y)
}

func validateProofHeader(proofHeader map[string]interface{}) error {
	// JOSE header parameters: typ, alg, jwk
	for _, k := range []string{"typ", "alg", "jwk"} {
		if _, ok := proofHeader[k]; !ok {
			return fmt.Errorf("missing DPoP proof JOSE header parameter %s", k)
		}
	}

	// 4. The typ JOSE Header Parameter has the value dpop+jwt.
	if proofHeader["typ"] != DpopTyp {
		return fmt.Errorf("DPoP proof typ JOSE header parameter mismatch")
	}

	// 5. The alg JOSE Header Parameter indicates a registered asymmetric
	//    digital signature algorithm [IANA.JOSE.ALGS], is not none,
	//    is supported by the application, and is acceptable per local policy.
	// Note: alg is already checked by jwt.ParseWithClaims()

	return nil
}

func validateProofClaims(proofClaims map[string]interface{}, req *http.Request, token string) error {
	// claims: jti, htm, htu, iat (, ath)
	for _, k := range []string{"jti", "htm", "htu", "iat", "ath"} {
		if _, ok := proofClaims[k]; !ok {
			return fmt.Errorf("missing DPoP proof claim %s", k)
		}
	}

	// 8. The htm claim matches the HTTP method of the current request.
	if proofClaims["htm"] != req.Method {
		return fmt.Errorf("DPoP proof htm claim mismatch")
	}

	if err := validateHtuClaim(proofClaims, req); err != nil {
		return err
	}

	// 10. If the server provided a nonce value to the client, the nonce
	//     claim matches the server-provided nonce value.
	// see also section
	// 9. Resource Server-Provided Nonce
	// Resource servers can also choose to provide a nonce value to be
	// included in DPoP proofs sent to them.
	//
	// So this is an optional feature. May be included later...

	if err := validateIatClaim(proofClaims); err != nil {
		return err
	}

	if err := validateAthClaim(proofClaims, token); err != nil {
		return err
	}

	return nil
}

func validateHtuClaim(proofClaims map[string]interface{}, req *http.Request) error {
	// 9. The htu claim matches the HTTP URI value for the HTTP request in
	//    which the JWT was received, ignoring any query and fragment parts.
	//
	// To reduce the likelihood of false negatives, servers SHOULD employ
	// syntax-based normalization (Section 6.2.2 of [RFC3986]) and scheme-
	// based normalization (Section 6.2.3 of [RFC3986]) before comparing the
	// htu claim.
	reqHtu, err := getReqHtu(req)
	if err != nil {
		return err
	}
	pcHtu, err := getPcHtu(proofClaims)
	if err != nil {
		return err
	}
	if pcHtu != reqHtu {
		return fmt.Errorf("DPoP proof htu claim mismatch")
	}

	return nil
}

func getReqHtu(req *http.Request) (string, error) {
	htu := &url.URL{
		Scheme: req.URL.Scheme,
		Host:   req.URL.Host,
		Path:   req.URL.Path,
	}
	var err error
	htu, err = normalize(htu)
	if err != nil {
		return "", err
	}

	return htu.String(), nil
}

func getPcHtu(proofClaims map[string]interface{}) (string, error) {
	htu, err := url.Parse(proofClaims["htu"].(string))
	if err != nil {
		return "", err
	}
	htu, err = normalize(htu)
	if err != nil {
		return "", err
	}

	return htu.String(), nil
}

func validateIatClaim(proofClaims map[string]interface{}) error {
	// 11. The creation time of the JWT, as determined by either the iat
	//     claim or a server managed timestamp via the nonce claim, is within
	//     an acceptable window (see Section 11.1).
	// acceptable window: 10s
	iatInt := int64(proofClaims["iat"].(float64))
	now := time.Now().Unix()
	if iatInt < now-10 {
		return fmt.Errorf("DPoP proof too old")
	}
	if iatInt > now+10 {
		return fmt.Errorf("DPoP proof too new")
	}

	return nil
}

func validateAthClaim(proofClaims map[string]interface{}, token string) error {
	// 12. If presented to a protected resource in conjunction with an access token,
	// (12.a) ensure that the value of the ath claim equals the hash of
	//      that access token, and
	hash := sha256.Sum256([]byte(token))
	ath := base64.RawURLEncoding.EncodeToString(hash[:])
	if proofClaims["ath"] != ath {
		return fmt.Errorf("DPoP proof ath claim mismatch")
	}

	return nil
}

func validateCnfClaim(tokenClaims, jwk map[string]interface{}) error {
	// (12.b) confirm that the public key to which the access token is
	//      bound matches the public key from the DPoP proof.
	cnf, ok := tokenClaims["cnf"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing DPoP access token cnf claim or wrong type")
	}
	atJkt, ok := cnf["jkt"].(string)
	if !ok {
		return fmt.Errorf("DPoP access token cnf claim missing jkt property or wrong type")
	}
	jkt := JwkToJKT(jwk)
	if atJkt != jkt {
		return fmt.Errorf("DPoP JWK thumbprint mismatch")
	}

	return nil
}

func toBigInt(bs []byte) *big.Int {
	return new(big.Int).SetBytes(bs)
}

func toInt(bs []byte) int {
	return int(toBigInt(bs).Int64())
}

func normalize(u *url.URL) (*url.URL, error) {
	// https://www.rfc-editor.org/rfc/rfc3986
	// 6.2.2.  Syntax-Based Normalization
	// 6.2.2.3.  Path Segment Normalization
	ru, err := url.Parse(u.RequestURI())
	if err != nil {
		return nil, err
	}
	// ResolveReference also resolves . and .. segements
	u = u.ResolveReference(ru)
	// 6.2.2.1.  Case Normalization
	// 6.2.2.2.  Percent-Encoding Normalization
	u.RawPath = u.Path

	// 6.2.3.  Scheme-Based Normalization
	hostname := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" ||
		u.Scheme == "http" && port == "80" ||
		u.Scheme == "https" && port == "443" {
		// remove : delimiter (and default port)
		u.Host = hostname
	} else {
		u.Host = hostname + ":" + port
	}
	return u, nil
}

/*
6.1 JWK Thumbprint Confirmation Method
 jkt:
    JWK SHA-256 Thumbprint confirmation method. The value of the jkt
		member MUST be the base64url encoding (as defined in [RFC7515]) of
		the JWK SHA-256 Thumbprint (according to [RFC7638]) of the DPoP
		public key (in JWK format) to which the access token is bound.
*/
// https://www.rfc-editor.org/rfc/rfc7638.html#section-3
// 3.  JSON Web Key (JWK) Thumbprint

// JwkToJKT creates a JWK SHA-256 thumbprint.
func JwkToJKT(jwk map[string]interface{}) string {
	jwks := JwkToString(jwk)
	//   2.  Hash the octets of the UTF-8 representation of this JSON object
	//       with a cryptographic hash function H.  For example, SHA-256 [SHS]
	//       might be used as H.  See Section 3.4 for a discussion on the
	//       choice of hash function.
	hash := sha256.Sum256([]byte(jwks))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

var requiredMembers = map[string]map[string]struct{}{
	"RSA": {
		"kty": {},
		"n":   {},
		"e":   {},
	},
	"ECDSA": {
		"kty": {},
		"crv": {},
		"x":   {},
		"y":   {},
	},
}

func JwkToString(jwk map[string]interface{}) string {
	rms := requiredMembers[jwk["kty"].(string)]
	//    1.  Construct a JSON object [RFC7159] containing only the required
	//        members of a JWK representing the key and with no whitespace or
	//        line breaks before or after any syntactic elements and with the
	//        required members ordered lexicographically by the Unicode
	//        [UNICODE] code points of the member names.  (This JSON object is
	//        itself a legal JWK representation of the key.)
	keys := make([]string, 0, len(jwk))
	for k := range jwk {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var entries []string
	for _, k := range keys {
		if _, ok := rms[k]; !ok {
			continue
		}
		entries = append(entries, fmt.Sprintf("%q:%q", k, jwk[k]))
	}
	return "{" + strings.Join(entries, ",") + "}"
}
