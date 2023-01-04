package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/rs/xid"

	acjwt "github.com/coupergateway/couper/accesscontrol/jwt"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/seetie"
)

const (
	clientSecretBasic = "client_secret_basic"
	clientSecretJwt   = "client_secret_jwt"
	clientSecretPost  = "client_secret_post"
	privateKeyJwt     = "private_key_jwt"
)

// Client represents an OAuth2 client.
type Client struct {
	authenticator *ClientAuthenticator
	backend       http.RoundTripper
	asConfig      config.OAuth2AS
	clientConfig  config.OAuth2Client
	grantType     string
}

func NewClient(evalCtx *hcl.EvalContext, grantType string, asConfig config.OAuth2AS, clientConfig config.OAuth2Client, backend http.RoundTripper) (*Client, error) {
	var authenticator *ClientAuthenticator
	if clientConfig.ClientAuthenticationRequired() {
		tokenEndpoint, err := asConfig.GetTokenEndpoint()
		if err != nil {
			return nil, err
		}
		authenticator, err = NewClientAuthenticator(evalCtx, clientConfig.GetTokenEndpointAuthMethod(), "token_endpoint_auth_method", clientConfig.GetClientID(), clientConfig.GetClientSecret(), tokenEndpoint, clientConfig.GetJWTSigningProfile())
		if err != nil {
			return nil, err
		}
	}

	return &Client{
		authenticator,
		backend,
		asConfig,
		clientConfig,
		grantType,
	}, nil
}

func (c *Client) requestToken(tokenReq *http.Request) ([]byte, int, error) {
	ctx, cancel := context.WithCancel(tokenReq.Context())
	defer cancel()

	tokenRes, err := c.backend.RoundTrip(tokenReq.WithContext(ctx))
	if err != nil {
		return nil, 0, err
	}
	defer tokenRes.Body.Close()

	tokenResBytes, err := io.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, tokenRes.StatusCode, err
	}

	return tokenResBytes, tokenRes.StatusCode, nil
}

func (c *Client) newTokenRequest(ctx context.Context, formParams url.Values) (*http.Request, error) {
	tokenURL, err := c.asConfig.GetTokenEndpoint()
	if err != nil {
		return nil, err
	}

	outreq, err := http.NewRequest(http.MethodPost, tokenURL, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Accept", "application/json")
	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	formParams.Set("grant_type", c.grantType)

	err = c.authenticator.Authenticate(&formParams, outreq)
	if err != nil {
		return nil, err
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	eval.SetBody(outreq, []byte(formParams.Encode()))

	return outreq.WithContext(outCtx), nil
}

func (c *Client) GetTokenResponse(ctx context.Context, formParams url.Values) (map[string]interface{}, string, error) {
	tokenReq, err := c.newTokenRequest(ctx, formParams)
	if err != nil {
		return nil, "", err
	}

	tokenResponse, statusCode, err := c.requestToken(tokenReq)
	if err != nil {
		return nil, "", err
	}

	tokenResponseData, accessToken, err := parseTokenResponse(tokenResponse)
	if err != nil {
		return nil, "", err
	}

	if statusCode != http.StatusOK {
		e, _ := tokenResponseData["error"].(string)
		msg := fmt.Sprintf("error=%s", e)
		errorDescription, dExists := tokenResponseData["error_description"].(string)
		if dExists {
			msg += fmt.Sprintf(", error_description=%s", errorDescription)
		}
		errorURI, uExists := tokenResponseData["error_uri"].(string)
		if uExists {
			msg += fmt.Sprintf(", error_uri=%s", errorURI)
		}
		return nil, "", fmt.Errorf("%s", msg)
	}

	return tokenResponseData, accessToken, nil
}

func parseTokenResponse(tokenResponse []byte) (map[string]interface{}, string, error) {
	var tokenResponseData map[string]interface{}

	err := json.Unmarshal(tokenResponse, &tokenResponseData)
	if err != nil {
		return nil, "", err
	}

	var accessToken string
	if t, ok := tokenResponseData["access_token"].(string); ok {
		accessToken = t
	} else {
		accessToken = ""
	}

	return tokenResponseData, accessToken, nil
}

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

// Authenticate authenticates an OAuth2 token or introspection request.
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
