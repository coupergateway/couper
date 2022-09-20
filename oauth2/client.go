package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	pkce "github.com/jimlambrt/go-oauth-pkce-code-verifier"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
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
	authnAud     string
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
		aud string
		key interface{}
		ttl int64
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

		switch authnMethod {
		case clientSecretBasic, clientSecretJwt, clientSecretPost:
			if clientConfig.GetClientSecret() == "" {
				return nil, fmt.Errorf("client_secret must not be empty")
			}
		default:
			// client_secret not needed
		}

		switch authnMethod {
		case clientSecretJwt, privateKeyJwt:
			dur, _, err := lib.CheckData(clientConfig.GetAuthnTTL(), clientConfig.GetAuthnSignatureAlgotithm())
			if err != nil {
				return nil, err
			}

			ttl = int64(dur.Seconds())
			var keyBytes []byte
			if authnMethod == privateKeyJwt {
				keyBytes, err = reader.ReadFromAttrFile("client authentication key", clientConfig.GetAuthnKey(), clientConfig.GetAuthnKeyFile())
				if err != nil {
					return nil, err
				}
			} else { // clientSecretJwt
				keyBytes = []byte(clientConfig.GetClientSecret())
			}

			key, err = lib.GetKey(keyBytes, clientConfig.GetAuthnSignatureAlgotithm())
			if err != nil {
				return nil, err
			}

			if aud = clientConfig.GetAuthnAudClaim(); aud == "" {
				aud, err = asConfig.GetTokenEndpoint()
				if err != nil {
					return nil, err
				}
			}
		default:
			// no key involved
		}
	}

	return &Client{
		backend,
		asConfig,
		clientConfig,
		grantType,
		authnMethod,
		aud,
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
		claims := jwt.MapClaims{
			"iss": c.clientConfig.GetClientID(),
			"sub": c.clientConfig.GetClientID(),
			"aud": c.authnAud,
			"iat": now,
			"exp": now + c.authnTTL,
			"jti": identifier.String(),
		}
		var headers map[string]interface{}
		if x5t := c.clientConfig.GetAuthnX5tHeader(); x5t != "" {
			headers = map[string]interface{}{
				"x5t": x5t,
			}
		}
		jwt, err := lib.CreateJWT(c.clientConfig.GetAuthnSignatureAlgotithm(), c.authnKey, claims, headers)
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
