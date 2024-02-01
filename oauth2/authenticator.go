package oauth2

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/rs/xid"

	acjwt "github.com/coupergateway/couper/accesscontrol/jwt"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/seetie"
)

// ClientAuthenticator is a component that authenticates OAuth2 token or introspection requests,
type ClientAuthenticator interface {
	Authenticate(formParams *url.Values, req *http.Request) error
}

// NewClientAuthenticator creates a new ClientAuthenticator.
func NewClientAuthenticator(evalCtx *hcl.EvalContext, authMethod *string, endpointAttrName, clientID, clientSecret, aud string, jwtSigningProfile *config.JWTSigningProfile) (ClientAuthenticator, error) {
	var authnMethod string
	if authMethod == nil {
		authnMethod = clientSecretBasic
	} else {
		authnMethod = *authMethod
	}

	if err := validate(authnMethod, endpointAttrName, clientID); err != nil {
		return nil, err
	}

	switch authnMethod {
	case clientSecretBasic:
		return newCsbClientAuthenticator(clientID, clientSecret, jwtSigningProfile)
	case clientSecretPost:
		return newCspClientAuthenticator(clientID, clientSecret, jwtSigningProfile)
	case clientSecretJwt:
		return newCsjClientAuthenticator(evalCtx, clientID, clientSecret, aud, jwtSigningProfile)
	default: // privateKeyJwt:
		return newPkjClientAuthenticator(evalCtx, clientID, clientSecret, aud, jwtSigningProfile)
	}
}

func validate(authnMethod, endpointAttrName, clientID string) error {
	switch authnMethod {
	case clientSecretBasic, clientSecretJwt, clientSecretPost, privateKeyJwt:
		// supported
	default:
		return fmt.Errorf("%s %q not supported", endpointAttrName, authnMethod)
	}

	if clientID == "" {
		return fmt.Errorf("client_id must not be empty")
	}

	return nil
}

type CsbClientAuthenticator struct {
	clientID     string
	clientSecret string
}

func newCsbClientAuthenticator(clientID, clientSecret string, jwtSigningProfile *config.JWTSigningProfile) (ClientAuthenticator, error) {
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret must not be empty with %s", clientSecretBasic)
	}
	if jwtSigningProfile != nil {
		return nil, fmt.Errorf("jwt_signing_profile block must not be set with %s", clientSecretBasic)
	}

	return &CsbClientAuthenticator{
		clientID,
		clientSecret,
	}, nil
}

func (ca *CsbClientAuthenticator) Authenticate(formParams *url.Values, req *http.Request) error {
	if ca == nil {
		return nil
	}

	req.SetBasicAuth(url.QueryEscape(ca.clientID), url.QueryEscape(ca.clientSecret))
	return nil
}

type CspClientAuthenticator struct {
	clientID     string
	clientSecret string
}

func newCspClientAuthenticator(clientID, clientSecret string, jwtSigningProfile *config.JWTSigningProfile) (ClientAuthenticator, error) {
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret must not be empty with %s", clientSecretPost)
	}
	if jwtSigningProfile != nil {
		return nil, fmt.Errorf("jwt_signing_profile block must not be set with %s", clientSecretPost)
	}

	return &CspClientAuthenticator{
		clientID,
		clientSecret,
	}, nil
}

func (ca *CspClientAuthenticator) Authenticate(formParams *url.Values, req *http.Request) error {
	if ca == nil {
		return nil
	}

	formParams.Set("client_id", ca.clientID)
	formParams.Set("client_secret", ca.clientSecret)
	return nil
}

type JwtClientAuthenticator struct {
	clientID string
	claims   map[string]interface{}
	headers  map[string]interface{}
	jsc      *lib.JWTSigningConfig
}

func csjAlgCheckFunc(algo acjwt.Algorithm) error {
	if !algo.IsHMAC() {
		return fmt.Errorf("inappropriate signature algorithm with %s", clientSecretJwt)
	}
	return nil
}

func newCsjClientAuthenticator(evalCtx *hcl.EvalContext, clientID, clientSecret, aud string, jwtSigningProfile *config.JWTSigningProfile) (ClientAuthenticator, error) {
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret must not be empty with %s", clientSecretJwt)
	}
	if err := validateSigningProfileCSJ(jwtSigningProfile); err != nil {
		return nil, err
	}
	jwtSigningProfile.Key = clientSecret

	signingConfig, headers, claims, err := getFromSigningProfile(evalCtx, clientID, aud, jwtSigningProfile, csjAlgCheckFunc)
	if err != nil {
		return nil, err
	}

	return &JwtClientAuthenticator{
		clientID,
		claims,
		headers,
		signingConfig,
	}, nil
}

func validateSigningProfileCSJ(jwtSigningProfile *config.JWTSigningProfile) error {
	if jwtSigningProfile == nil {
		return fmt.Errorf("jwt_signing_profile block must be set with %s", clientSecretJwt)
	}
	if jwtSigningProfile.Key != "" {
		return fmt.Errorf("key must not be set with %s", clientSecretJwt)
	}
	if jwtSigningProfile.KeyFile != "" {
		return fmt.Errorf("key_file must not be set with %s", clientSecretJwt)
	}
	return nil
}

func pkjAlgCheckFunc(algo acjwt.Algorithm) error {
	if algo.IsHMAC() {
		return fmt.Errorf("inappropriate signature algorithm with %s", privateKeyJwt)
	}
	return nil
}

func newPkjClientAuthenticator(evalCtx *hcl.EvalContext, clientID, clientSecret, aud string, jwtSigningProfile *config.JWTSigningProfile) (ClientAuthenticator, error) {
	if clientSecret != "" {
		return nil, fmt.Errorf("client_secret must not be set with %s", privateKeyJwt)
	}
	if err := validateSigningProfilePKJ(jwtSigningProfile); err != nil {
		return nil, err
	}

	signingConfig, headers, claims, err := getFromSigningProfile(evalCtx, clientID, aud, jwtSigningProfile, pkjAlgCheckFunc)
	if err != nil {
		return nil, err
	}

	return &JwtClientAuthenticator{
		clientID,
		claims,
		headers,
		signingConfig,
	}, nil
}

func validateSigningProfilePKJ(jwtSigningProfile *config.JWTSigningProfile) error {
	if jwtSigningProfile == nil {
		return fmt.Errorf("jwt_signing_profile block must be set with %s", privateKeyJwt)
	}
	if jwtSigningProfile.Key == "" && jwtSigningProfile.KeyFile == "" {
		return fmt.Errorf("key and key_file must not both be empty with %s", privateKeyJwt)
	}
	return nil
}

func getFromSigningProfile(evalCtx *hcl.EvalContext, clientID, aud string, jwtSigningProfile *config.JWTSigningProfile, algCheckFunc func(algo acjwt.Algorithm) error) (signingConfig *lib.JWTSigningConfig, headers, claims map[string]interface{}, err error) {
	signingConfig, err = lib.NewJWTSigningConfigFromJWTSigningProfile(jwtSigningProfile, algCheckFunc)
	if err != nil {
		return nil, nil, nil, err
	}

	headers, err = getHeadersFromSigningConfig(evalCtx, signingConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	claims, err = getClaimsFromSigningConfig(evalCtx, signingConfig, aud, clientID)
	if err != nil {
		return nil, nil, nil, err
	}

	return signingConfig, headers, claims, nil
}

func getHeadersFromSigningConfig(evalCtx *hcl.EvalContext, signingConfig *lib.JWTSigningConfig) (map[string]interface{}, error) {
	var headers map[string]interface{}
	if signingConfig.Headers != nil {
		v, err := eval.Value(evalCtx, signingConfig.Headers)
		if err != nil {
			return nil, err
		}
		headers = seetie.ValueToMap(v)
		if _, exists := headers["alg"]; exists {
			return nil, fmt.Errorf(`"alg" cannot be set via "headers"`)
		}
	}
	return headers, nil
}

func getClaimsFromSigningConfig(evalCtx *hcl.EvalContext, signingConfig *lib.JWTSigningConfig, aud, clientID string) (map[string]interface{}, error) {
	var claims map[string]interface{}
	if signingConfig.Claims != nil {
		cl, err := eval.Value(evalCtx, signingConfig.Claims)
		if err != nil {
			return nil, err
		}
		claims = seetie.ValueToMap(cl)
	} else {
		claims = make(map[string]interface{}, 3)
	}
	if _, set := claims["aud"]; !set && aud != "" {
		claims["aud"] = aud
	}
	claims["iss"] = clientID
	claims["sub"] = clientID
	return claims, nil
}

func (ca *JwtClientAuthenticator) Authenticate(formParams *url.Values, req *http.Request) error {
	if ca == nil {
		return nil
	}

	claims := make(map[string]interface{})
	for k, v := range ca.claims {
		claims[k] = v
	}
	now := time.Now().Unix()
	claims["iat"] = now
	claims["exp"] = now + ca.jsc.TTL
	// Create an identifier as per RFC 7519 section 4.1.7:
	// The identifier value MUST be assigned in a manner that ensures that
	// there is a negligible probability that the same value will be
	// accidentally assigned to a different data object
	claims["jti"] = "client_assertion-" + xid.New().String()
	clientAssertion, err := lib.CreateJWT(ca.jsc.SignatureAlgorithm, ca.jsc.Key, claims, ca.headers)
	if err != nil {
		return err
	}
	formParams.Set("client_id", ca.clientID)
	formParams.Set("client_assertion", clientAssertion)
	formParams.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	return nil
}
