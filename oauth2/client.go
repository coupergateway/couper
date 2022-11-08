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
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"

	acjwt "github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

const (
	clientSecretBasic = "client_secret_basic"
	clientSecretJwt   = "client_secret_jwt"
	clientSecretPost  = "client_secret_post"
	privateKeyJwt     = "private_key_jwt"
)

// Client represents an OAuth2 client.
type Client struct {
	backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
	grantType    string
	authnMethod  string
	authnJSC     *lib.JWTSigningConfig
	authnClaims  map[string]interface{}
	authnHeaders map[string]interface{}
}

func NewClient(evalCtx *hcl.EvalContext, grantType string, asConfig config.OAuth2AS, clientConfig config.OAuth2Client, backend http.RoundTripper) (*Client, error) {
	var authnMethod string
	teAuthMethod := clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod == nil {
		authnMethod = clientSecretBasic
	} else {
		authnMethod = *teAuthMethod
	}

	var (
		signingConfig *lib.JWTSigningConfig
		claims        map[string]interface{}
		headers       map[string]interface{}
	)
	switch authnMethod {
	case clientSecretBasic, clientSecretJwt, clientSecretPost, privateKeyJwt:
		// supported
	default:
		return nil, fmt.Errorf("token_endpoint_auth_method %q not supported", *teAuthMethod)
	}

	if clientConfig.ClientAuthenticationRequired() {
		clientID := clientConfig.GetClientID()
		if clientID == "" {
			return nil, fmt.Errorf("client_id must not be empty")
		}

		clientSecret := clientConfig.GetClientSecret()

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

		jwtSigningProfile := clientConfig.GetJWTSigningProfile()
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
			if _, set := claims["aud"]; !set {
				tokenEndpoint, err := asConfig.GetTokenEndpoint()
				if err != nil {
					return nil, err
				}
				claims["aud"] = tokenEndpoint
			}
			claims["iss"] = clientID
			claims["sub"] = clientID
		default:
			if jwtSigningProfile != nil {
				return nil, fmt.Errorf("jwt_signing_profile block must not be set with %s", authnMethod)
			}
		}
	}

	return &Client{
		backend,
		asConfig,
		clientConfig,
		grantType,
		authnMethod,
		signingConfig,
		claims,
		headers,
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

	err = c.authenticateClient(&formParams, outreq)
	if err != nil {
		return nil, err
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	eval.SetBody(outreq, []byte(formParams.Encode()))

	return outreq.WithContext(outCtx), nil
}

func (c *Client) authenticateClient(formParams *url.Values, tokenReq *http.Request) error {
	if !c.clientConfig.ClientAuthenticationRequired() {
		return nil
	}

	clientID := c.clientConfig.GetClientID()
	clientSecret := c.clientConfig.GetClientSecret()
	switch c.authnMethod {
	case clientSecretBasic:
		tokenReq.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	case clientSecretPost:
		formParams.Set("client_id", clientID)
		formParams.Set("client_secret", clientSecret)
	case clientSecretJwt, privateKeyJwt:
		// Although this is unrelated to PKCE, it is used to create an identifier as per RFC 7519 section 4.1.7:
		// The identifier value MUST be assigned in a manner that ensures that
		// there is a negligible probability that the same value will be
		// accidentally assigned to a different data object
		identifier, err := pkce.CreateCodeVerifier()
		if err != nil {
			return err
		}

		claims := make(map[string]interface{})
		for k, v := range c.authnClaims {
			claims[k] = v
		}
		now := time.Now().Unix()
		claims["iat"] = now
		claims["exp"] = now + c.authnJSC.TTL
		claims["jti"] = identifier.String()
		clientAssertion, err := lib.CreateJWT(c.authnJSC.SignatureAlgorithm, c.authnJSC.Key, claims, c.authnHeaders)
		if err != nil {
			return err
		}
		formParams.Set("client_id", clientID)
		formParams.Set("client_assertion", clientAssertion)
		formParams.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	default:
		// already handled with error
	}
	return nil
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
		return nil, "", fmt.Errorf(msg)
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
