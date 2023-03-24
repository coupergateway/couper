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
type ClientAuthenticator struct {
	authnMethod  string
	claims       map[string]interface{}
	clientID     string
	clientSecret string
	headers      map[string]interface{}
	jsc          *lib.JWTSigningConfig
}

// NewClientAuthenticator creates a new ClientAuthenticator.
func NewClientAuthenticator(evalCtx *hcl.EvalContext, authMethod *string, endpointAttrName, clientID, clientSecret, aud string, jwtSigningProfile *config.JWTSigningProfile) (*ClientAuthenticator, error) {
	var authnMethod string
	if authMethod == nil {
		authnMethod = clientSecretBasic
	} else {
		authnMethod = *authMethod
	}

	switch authnMethod {
	case clientSecretBasic, clientSecretJwt, clientSecretPost, privateKeyJwt:
		// supported
	default:
		return nil, fmt.Errorf("%s %q not supported", endpointAttrName, authnMethod)
	}

	if clientID == "" {
		return nil, fmt.Errorf("client_id must not be empty")
	}

	switch authnMethod {
	case clientSecretBasic, clientSecretJwt, clientSecretPost:
		if clientSecret == "" {
			return nil, fmt.Errorf("client_secret must not be empty with %s", authnMethod)
		}
	default: // privateKeyJwt
		if clientSecret != "" {
			return nil, fmt.Errorf("client_secret must not be set with %s", authnMethod)
		}
	}

	var (
		claims        map[string]interface{}
		headers       map[string]interface{}
		signingConfig *lib.JWTSigningConfig
	)

	switch authnMethod {
	case clientSecretJwt, privateKeyJwt:
		if jwtSigningProfile == nil {
			return nil, fmt.Errorf("jwt_signing_profile block must be set with %s", authnMethod)
		}
		if authnMethod == privateKeyJwt {
			if jwtSigningProfile.Key == "" && jwtSigningProfile.KeyFile == "" {
				return nil, fmt.Errorf("key and key_file must not both be empty with %s", authnMethod)
			}
		} else { // clientSecretJwt
			if jwtSigningProfile.Key != "" {
				return nil, fmt.Errorf("key must not be set with %s", authnMethod)
			}
			if jwtSigningProfile.KeyFile != "" {
				return nil, fmt.Errorf("key_file must not be set with %s", authnMethod)
			}
			jwtSigningProfile.Key = clientSecret
		}

		algCheckFunc := func(algo acjwt.Algorithm) error {
			if authnMethod == clientSecretJwt && !algo.IsHMAC() || authnMethod == privateKeyJwt && algo.IsHMAC() {
				return fmt.Errorf("inappropriate signature algorithm with %s", authnMethod)
			}
			return nil
		}
		var err error
		signingConfig, err = lib.NewJWTSigningConfigFromJWTSigningProfile(jwtSigningProfile, algCheckFunc)
		if err != nil {
			return nil, err
		}

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

		// get claims from signing profile
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
	default:
		if jwtSigningProfile != nil {
			return nil, fmt.Errorf("jwt_signing_profile block must not be set with %s", authnMethod)
		}
	}

	return &ClientAuthenticator{
		authnMethod,
		claims,
		clientID,
		clientSecret,
		headers,
		signingConfig,
	}, nil
}

// Authenticate authenticates an OAuth2 token or introspection request by adding necessary form parameters/header fields.
func (ca *ClientAuthenticator) Authenticate(formParams *url.Values, tokenReq *http.Request) error {
	if ca == nil {
		return nil
	}

	switch ca.authnMethod {
	case clientSecretBasic:
		tokenReq.SetBasicAuth(url.QueryEscape(ca.clientID), url.QueryEscape(ca.clientSecret))
	case clientSecretPost:
		formParams.Set("client_id", ca.clientID)
		formParams.Set("client_secret", ca.clientSecret)
	case clientSecretJwt, privateKeyJwt:
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
	default:
		// already handled with error
	}
	return nil
}
