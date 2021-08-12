package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/content"
	"github.com/avenga/couper/eval/lib"
)

// Client represents the OAuth2 client.
type Client struct {
	Backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
}

// NewOAuth2CC creates a new OAuth2 Client Credentials client.
func NewOAuth2CC(conf *config.OAuth2ReqAuth, backend http.RoundTripper) (*Client, error) {
	backendErr := errors.Backend.Label(conf.Reference())
	if grantType := conf.GrantType; grantType != "client_credentials" {
		return nil, backendErr.Messagef("grant_type %s not supported", grantType)
	}

	if teAuthMethod := conf.TokenEndpointAuthMethod; teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, backendErr.Messagef("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	return &Client{backend, conf, conf}, nil
}

func (c *Client) requestToken(ctx context.Context, requestParams map[string]string) ([]byte, error) {
	tokenReq, err := c.newTokenRequest(ctx, requestParams)
	if err != nil {
		return nil, err
	}

	tokenRes, err := c.Backend.RoundTrip(tokenReq)
	if err != nil {
		return nil, err
	}

	tokenResBytes, err := io.ReadAll(tokenRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(c.asConfig.Reference()).Message("token request read error").With(err)
	}

	if tokenRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(c.asConfig.Reference()).Messagef("token request failed, response=%q", string(tokenResBytes))
	}

	return tokenResBytes, nil
}

func (c *Client) newTokenRequest(ctx context.Context, requestParams map[string]string) (*http.Request, error) {
	post := url.Values{}
	grantType := c.clientConfig.GetGrantType()
	post.Set("grant_type", grantType)

	if scope := c.clientConfig.GetScope(); scope != "" && grantType != "authorization_code" {
		post.Set("scope", scope)
	}
	if requestParams != nil {
		for key, value := range requestParams {
			post.Set(key, value)
		}
	}
	teAuthMethod := c.clientConfig.GetTokenEndpointAuthMethod()
	if teAuthMethod != nil && *teAuthMethod == "client_secret_post" {
		post.Set("client_id", c.clientConfig.GetClientID())
		post.Set("client_secret", c.clientConfig.GetClientSecret())
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodPost, "", nil)
	if err != nil {
		return nil, err
	}

	eval.SetBody(outreq, []byte(post.Encode()))

	outreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if teAuthMethod == nil || *teAuthMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.clientConfig.GetClientID() + ":" + c.clientConfig.GetClientSecret()))

		outreq.Header.Set("Authorization", "Basic "+auth)
	}

	outCtx := context.WithValue(ctx, request.TokenRequest, "oauth2")

	tokenURL, err := c.asConfig.GetTokenEndpoint()
	if err != nil {
		return nil, err
	}

	if tokenURL != "" {
		outCtx = context.WithValue(outCtx, request.URLAttribute, tokenURL)
	}

	return outreq.WithContext(outCtx), nil
}

func (c *Client) getTokenResponse(ctx context.Context, requestParams map[string]string) ([]byte, map[string]interface{}, string, error) {
	tokenResponse, err := c.requestToken(ctx, requestParams)
	if err != nil {
		return nil, nil, "", err
	}

	tokenResponseData, accessToken, err := ParseTokenResponse(tokenResponse)
	if err != nil {
		return nil, nil, "", errors.Oauth2.Messagef("parsing token response JSON failed, response=%q", string(tokenResponse)).With(err)
	}

	return tokenResponse, tokenResponseData, accessToken, nil
}

func (c *Client) GetTokenResponse(ctx context.Context) ([]byte, map[string]interface{}, string, error) {
	return c.getTokenResponse(ctx, nil)
}

// AcClient represents an OAuth2 client using the authorization code flow.
type AcClient interface {
	GetName() string
	GetTokenResponse(ctx context.Context, callbackURL *url.URL) ([]byte, map[string]interface{}, string, error)
	validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error
}

type AbstractAcClient struct {
	AcClient
	Client
	name string
}

func (a AbstractAcClient) GetName() string {
	return a.name
}

func (a AbstractAcClient) GetTokenResponse(ctx context.Context, callbackURL *url.URL) ([]byte, map[string]interface{}, string, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, nil, "", errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	callbackURLVal, err := content.GetContextAttribute(a.clientConfig.HCLBody(), ctx, lib.CallbackURL)
	if callbackURLVal == "" {
		return nil, nil, "", fmt.Errorf("%s is required", callbackURL)
	}
	absoluteURL, err := lib.AbsoluteURL(callbackURLVal, eval.NewRawOrigin(callbackURL))
	if err != nil {
		return nil, nil, "", err
	}

	requestParams := map[string]string{
		"code":         code,
		"redirect_uri": absoluteURL,
	}

	verifierVal, err := content.GetContextAttribute(a.clientConfig.HCLBody(), ctx, "verifier_value")
	verifierValue := strings.TrimSpace(verifierVal)
	if verifierValue == "" {
		return nil, nil, "", errors.Oauth2.Message("Empty verifier_value")
	}

	verifierMethod, err := getVerifierMethod(a.asConfig)
	if err != nil {
		return nil, nil, "", err
	}

	var hashedVerifierValue string
	if verifierMethod == config.CcmS256 {
		requestParams["code_verifier"] = verifierValue
	} else {
		hashedVerifierValue = Base64urlSha256(verifierValue)
	}

	if verifierMethod == "state" {
		stateFromParam := query.Get("state")
		if stateFromParam == "" {
			return nil, nil, "", errors.Oauth2.Messagef("missing state query parameter; query=%q", callbackURL.RawQuery)
		}

		if hashedVerifierValue != stateFromParam {
			return nil, nil, "", errors.Oauth2.Messagef("state mismatch: %q (from query param) vs. %q (verifier_value: %q)", stateFromParam, hashedVerifierValue, verifierValue)
		}
	}

	tokenResponse, tokenResponseData, accessToken, err := a.getTokenResponse(ctx, requestParams)
	if err != nil {
		return nil, nil, "", err
	}

	if err = a.validateTokenResponseData(ctx, tokenResponseData, hashedVerifierValue, verifierValue, accessToken); err != nil {
		return nil, nil, "", err
	}

	return tokenResponse, tokenResponseData, accessToken, nil
}

// OAuth2AcClient represents an OAuth2 client using the (plain) authorization code flow.
type OAuth2AcClient struct {
	*AbstractAcClient
}

// NewOAuth2AC creates a new OAuth2 Authorization Code client.
func NewOAuth2AC(acClientConf config.OAuth2AcClient, oauth2AsConf config.OAuth2AS, backend http.RoundTripper) (*OAuth2AcClient, error) {
	if grantType := acClientConf.GetGrantType(); grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	if teAuthMethod := acClientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}

	verifierMethod, err := acClientConf.GetVerifierMethod()
	if err != nil {
		return nil, err
	}

	if verifierMethod != config.CcmS256 && verifierMethod != "nonce" && verifierMethod != "state" {
		return nil, fmt.Errorf("verifier_method %s not supported", verifierMethod)
	}

	client := Client{
		Backend:      backend,
		asConfig:     oauth2AsConf,
		clientConfig: acClientConf,
	}
	o := &OAuth2AcClient{&AbstractAcClient{Client: client, name: acClientConf.GetName()}}
	o.AcClient = o
	return o, nil
}

func (o *OAuth2AcClient) validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error {
	return nil
}

func ParseTokenResponse(tokenResponse []byte) (map[string]interface{}, string, error) {
	var tokenResponseData map[string]interface{}

	err := json.Unmarshal(tokenResponse, &tokenResponseData)
	if err != nil {
		return nil, "", err
	}

	var accessToken string
	if t, ok := tokenResponseData["access_token"].(string); ok {
		accessToken = t
	} else {
		return nil, "", fmt.Errorf("missing access token")
	}

	return tokenResponseData, accessToken, nil
}

func Base64urlSha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func getVerifierMethod(conf interface{}) (string, error) {
	clientConf, ok := conf.(config.OAuth2AcClient)
	if !ok {
		return "", fmt.Errorf("could not obtain verifier method configuration")
	}
	return clientConf.GetVerifierMethod()
}
