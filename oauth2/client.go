package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v4"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/reader"
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
	authnAlgo    string
	authnClaims  jwt.MapClaims
	authnHeaders map[string]interface{}
	authnKey     interface{}
	authnTTL     int64
}

func NewClient(grantType string, asConfig config.OAuth2AS, clientConfig config.OAuth2Client, backend http.RoundTripper) (*Client, error) {
	var authnMethod string
	teAuthMethod := clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod == nil {
		authnMethod = clientSecretBasic
	} else {
		authnMethod = *teAuthMethod
	}

	var (
		algorithm string
		claims    jwt.MapClaims
		headers   map[string]interface{}
		key       interface{}
		ttl       int64
	)
	switch authnMethod {
	case clientSecretBasic, clientSecretJwt, clientSecretPost, privateKeyJwt:
		// supported
	default:
		return nil, fmt.Errorf("token_endpoint_auth_method %q not supported", *teAuthMethod)
	}

	if clientConfig.ClientAuthenticationRequired() {
		if clientConfig.GetClientID() == "" {
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
			dur, algo, err := lib.CheckData(jwtSigningProfile.TTL, jwtSigningProfile.SignatureAlgorithm)
			if err != nil {
				return nil, err
			}
			if authnMethod == clientSecretJwt && !algo.IsHMAC() || authnMethod == privateKeyJwt && algo.IsHMAC() {
				return nil, fmt.Errorf("inappropriate signature algorithm with %s", authnMethod)
			}

			ttl = int64(dur.Seconds())
			var keyBytes []byte
			if authnMethod == privateKeyJwt {
				if jwtSigningProfile.Key == "" && jwtSigningProfile.KeyFile == "" {
					return nil, fmt.Errorf("key and key_file must not both be empty with %s", authnMethod)
				}
				keyBytes, err = reader.ReadFromAttrFile("client authentication key", jwtSigningProfile.Key, jwtSigningProfile.KeyFile)
				if err != nil {
					return nil, err
				}
			} else { // clientSecretJwt
				keyBytes = []byte(clientSecret)
				if jwtSigningProfile.Key != "" {
					return nil, fmt.Errorf("key must not be set with %s", authnMethod)
				}
				if jwtSigningProfile.KeyFile != "" {
					return nil, fmt.Errorf("key_file must not be set with %s", authnMethod)
				}
			}

			key, err = lib.GetKey(keyBytes, jwtSigningProfile.SignatureAlgorithm)
			if err != nil {
				return nil, err
			}
			algorithm = jwtSigningProfile.SignatureAlgorithm

			if jwtSigningProfile.Headers != nil {
				v, err := eval.Value(nil, jwtSigningProfile.Headers)
				if err != nil {
					return nil, err
				}
				headers = seetie.ValueToMap(v)
			}

			tokenEndpoint, err := asConfig.GetTokenEndpoint()
			if err != nil {
				return nil, err
			}
			claims = jwt.MapClaims{
				// default audience
				"aud": tokenEndpoint,
			}
			// get claims from signing profile
			if jwtSigningProfile.Claims != nil {
				cl, err := eval.Value(nil, jwtSigningProfile.Claims)
				if err != nil {
					return nil, err
				}
				for k, v := range seetie.ValueToMap(cl) {
					claims[k] = v
				}
			}
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
		algorithm,
		claims,
		headers,
		key,
		ttl,
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

		now := time.Now().Unix()
		c.authnClaims["iss"] = c.clientConfig.GetClientID()
		c.authnClaims["sub"] = c.clientConfig.GetClientID()
		c.authnClaims["iat"] = now
		c.authnClaims["exp"] = now + c.authnTTL
		c.authnClaims["jti"] = identifier.String()
		jwt, err := lib.CreateJWT(c.authnAlgo, c.authnKey, c.authnClaims, c.authnHeaders)
		if err != nil {
			return err
		}
		formParams.Set("client_id", clientID)
		formParams.Set("client_assertion", jwt)
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
