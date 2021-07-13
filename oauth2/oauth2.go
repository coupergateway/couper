package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
)

// Client represents the OAuth2 client.
type Client struct {
	Backend      http.RoundTripper
	asConfig     config.OAuth2AS
	clientConfig config.OAuth2Client
}

// NewOAuth2CC creates a new OAuth2 Client Credentials client.
func NewOAuth2CC(clientConf config.OAuth2Client, asConf config.OAuth2AS, backend http.RoundTripper) (*Client, error) {
	backendErr := errors.Backend.Label(asConf.Reference())
	if grantType := clientConf.GetGrantType(); grantType != "client_credentials" {
		return nil, backendErr.Messagef("grant_type %s not supported", grantType)
	}

	if teAuthMethod := clientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, backendErr.Messagef("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	return &Client{backend, asConf, clientConf}, nil
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

	tokenResBytes, err := ioutil.ReadAll(tokenRes.Body)
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

	if scope := c.clientConfig.GetScope(); scope != nil && grantType != "authorization_code" {
		post.Set("scope", *scope)
	}
	if acClientConfig, ok := c.clientConfig.(config.OAuth2AcClient); ok && grantType == "authorization_code" {
		post.Set("redirect_uri", *acClientConfig.GetRedirectURI())
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

	if tokenURL := c.asConfig.GetTokenEndpoint(); tokenURL != "" {
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

// Client represents the OAuth2 client using the authorization code flow.
type AcClient struct {
	Client
	jwtParser *jwt.Parser
}

// NewOAuth2AC creates a new OAuth2 Authorization Code client.
func NewOAuth2AC(acClientConf config.OAuth2AcClient, oidcAsConf config.OidcAS, backend http.RoundTripper) (*AcClient, error) {
	if grantType := acClientConf.GetGrantType(); grantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type %s not supported", grantType)
	}

	if teAuthMethod := acClientConf.GetTokenEndpointAuthMethod(); teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, fmt.Errorf("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	csrf := acClientConf.GetCsrf()
	pkce := acClientConf.GetPkce()
	if pkce == nil && csrf == nil {
		return nil, fmt.Errorf("CSRF protection not configured")
	}
	if csrf != nil {
		if csrf.TokenParam != "state" && csrf.TokenParam != "nonce" {
			return nil, fmt.Errorf("csrf_token_param %s not supported", csrf.TokenParam)
		}
		content, _, diags := csrf.HCLBody().PartialContent(csrf.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		csrf.Content = content
	}
	if pkce != nil {
		if pkce.CodeChallengeMethod != lib.CcmPlain && pkce.CodeChallengeMethod != lib.CcmS256 {
			return nil, fmt.Errorf("code_challenge_method %s not supported", pkce.CodeChallengeMethod)
		}
		content, _, diags := pkce.HCLBody().PartialContent(pkce.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		pkce.Content = content
	}

	options := []jwt.ParserOption{
		// jwt.WithValidMethods([]string{algo.String()}),
		jwt.WithLeeway(time.Second),
		// 2. The Issuer Identifier for the OpenID Provider (which is typically
		//    obtained during Discovery) MUST exactly match the value of the iss
		//    (issuer) Claim.
		jwt.WithIssuer(oidcAsConf.GetIssuer()),
		// 3. The Client MUST validate that the aud (audience) Claim contains its
		//    client_id value registered at the Issuer identified by the iss
		//    (issuer) Claim as an audience. The aud (audience) Claim MAY contain
		//    an array with more than one element. The ID Token MUST be rejected if
		//    the ID Token does not list the Client as a valid audience, or if it
		//    contains additional audiences not trusted by the Client.
		jwt.WithAudience(acClientConf.GetClientID()),
	}
	jwtParser := jwt.NewParser(options...)

	client := Client{
		Backend:      backend,
		asConfig:     oidcAsConf,
		clientConfig: acClientConf,
	}
	return &AcClient{client, jwtParser}, nil
}

func (a *AcClient) GetAcClientConfig() config.OAuth2AcClient {
	acClientConfig, _ := a.clientConfig.(config.OAuth2AcClient)
	return acClientConfig
}

func (a *AcClient) getOidcAsConfig() config.OidcAS {
	oidcAsConfig, _ := a.asConfig.(config.OidcAS)
	return oidcAsConfig
}

func (a *AcClient) GetTokenResponse(ctx context.Context, callbackURL *url.URL) ([]byte, map[string]interface{}, string, error) {
	query := callbackURL.Query()
	code := query.Get("code")
	if code == "" {
		return nil, nil, "", errors.Oauth2.Messagef("missing code query parameter; query=%q", callbackURL.RawQuery)
	}

	requestParams := map[string]string{"code": code}

	evalContext, _ := ctx.Value(eval.ContextType).(*eval.Context)

	if pkce := a.GetAcClientConfig().GetPkce(); pkce != nil {
		v, _ := pkce.Content.Attributes["code_verifier_value"]
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		codeVerifierValue := strings.TrimSpace(seetie.ValueToString(ctyVal))
		if codeVerifierValue == "" {
			return nil, nil, "", errors.Oauth2.Message("Empty PKCE code_verifier_value")
		}
		requestParams["code_verifier"] = codeVerifierValue
	}

	var csrfToken, csrfTokenValue string
	if csrf := a.GetAcClientConfig().GetCsrf(); csrf != nil {
		v, _ := csrf.Content.Attributes["token_value"]
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		csrfTokenValue = strings.TrimSpace(seetie.ValueToString(ctyVal))
		if csrfTokenValue == "" {
			return nil, nil, "", errors.Oauth2.Message("Empty CSRF token_value")
		}
		csrfToken = Base64urlSha256(csrfTokenValue)

		// validate state param value against CSRF token
		if csrf.TokenParam == "state" {
			csrfTokenFromParam := query.Get(csrf.TokenParam)
			if csrfTokenFromParam == "" {
				return nil, nil, "", errors.Oauth2.Messagef("missing state query parameter; query=%q", callbackURL.RawQuery)
			}

			if csrfToken != csrfTokenFromParam {
				return nil, nil, "", errors.Oauth2.Messagef("CSRF token mismatch: %q (from query param) vs. %q (s256: %q)", csrfTokenFromParam, csrfTokenValue, csrfToken)
			}
		}
	}

	tokenResponse, tokenResponseData, accessToken, err := a.getTokenResponse(ctx, requestParams)
	if err != nil {
		return nil, nil, "", err
	}

	if idTokenString, ok := tokenResponseData["id_token"].(string); ok {
		idToken, _, err := a.jwtParser.ParseUnverified(idTokenString, jwt.MapClaims{})
		if err != nil {
			return nil, nil, "", err
		}

		// 2.  ID Token
		// iss
		// 		REQUIRED.
		// aud
		// 		REQUIRED.
		// 3.1.3.7.  ID Token Validation
		// 3. The Client MUST validate that the aud (audience) Claim contains
		//    its client_id value registered at the Issuer identified by the
		//    iss (issuer) Claim as an audience. The aud (audience) Claim MAY
		//    contain an array with more than one element. The ID Token MUST
		//    be rejected if the ID Token does not list the Client as a valid
		//    audience, or if it contains additional audiences not trusted by
		//    the Client.
		if err := idToken.Claims.Valid(a.jwtParser.ValidationHelper); err != nil {
			return nil, nil, "", err
		}

		idtc, userinfo, err := a.validateIdTokenClaims(ctx, idToken.Claims, csrfToken, csrfTokenValue, accessToken)
		if err != nil {
			return nil, nil, "", err
		}

		tokenResponseData["id_token_claims"] = idtc
		tokenResponseData["userinfo"] = userinfo
	}

	return tokenResponse, tokenResponseData, accessToken, nil
}

func (a *AcClient) validateIdTokenClaims(ctx context.Context, claims jwt.Claims, csrfToken, csrfTokenValue string, accessToken string) (map[string]interface{}, map[string]interface{}, error) {
	var idTokenClaims jwt.MapClaims
	if tc, ok := claims.(jwt.MapClaims); ok {
		idTokenClaims = tc
	}

	// 2.  ID Token
	// exp
	// 		REQUIRED.
	if _, expExists := idTokenClaims["exp"]; !expExists {
		return nil, nil, errors.Oauth2.Messagef("missing exp claim in ID token, claims='%#v'", idTokenClaims)
	}
	// iat
	// 		REQUIRED.
	if _, iatExists := idTokenClaims["iat"]; !iatExists {
		return nil, nil, errors.Oauth2.Messagef("missing iat claim in ID token, claims='%#v'", idTokenClaims)
	}

	// 3.1.3.7.  ID Token Validation
	// 4. If the ID Token contains multiple audiences, the Client SHOULD verify
	//    that an azp Claim is present.
	azp, azpExists := idTokenClaims["azp"]
	if auds, audsOK := idTokenClaims["aud"].([]interface{}); audsOK && len(auds) > 1 && !azpExists {
		return nil, nil, errors.Oauth2.Messagef("missing azp claim in ID token, claims='%#v'", idTokenClaims)
	}
	// 5. If an azp (authorized party) Claim is present, the Client SHOULD
	//    verify that its client_id is the Claim Value.
	if azpExists && azp != a.clientConfig.GetClientID() {
		return nil, nil, errors.Oauth2.Messagef("azp claim / client ID mismatch, azp = %q, client ID = %q", azp, a.clientConfig.GetClientID())
	}

	// validate nonce claim value against CSRF token
	csrf := a.GetAcClientConfig().GetCsrf()
	if csrf != nil && csrf.TokenParam == "nonce" {
		// 11. If a nonce value was sent in the Authentication Request, a nonce
		//     Claim MUST be present and its value checked to verify that it is the
		//     same value as the one that was sent in the Authentication Request.
		//     The Client SHOULD check the nonce value for replay attacks. The
		//     precise method for detecting replay attacks is Client specific.
		var nonce string
		if n, ok := idTokenClaims["nonce"].(string); ok {
			nonce = n
		} else {
			return nil, nil, errors.Oauth2.Messagef("missing nonce claim in ID token, claims='%#v'", idTokenClaims)
		}

		if csrfToken != nonce {
			return nil, nil, errors.Oauth2.Messagef("CSRF token mismatch: %q (from nonce claim) vs. %q (s256: %q)", nonce, csrfTokenValue, csrfToken)
		}
	}

	// 2.  ID Token
	// sub
	// 		REQUIRED.
	var subIdtoken string
	if s, ok := idTokenClaims["sub"].(string); ok {
		subIdtoken = s
	} else {
		return nil, nil, errors.Oauth2.Messagef("missing sub claim in ID token, claims='%#v'", idTokenClaims)
	}

	userinfoResponse, err := a.requestUserinfo(ctx, accessToken)
	if err != nil {
		return nil, nil, err
	}

	userinfoResponseString := string(userinfoResponse)
	var userinfoData map[string]interface{}
	err = json.Unmarshal(userinfoResponse, &userinfoData)
	if err != nil {
		return nil, nil, errors.Oauth2.Messagef("parsing userinfo response JSON failed, response=%q", userinfoResponseString).With(err)
	}

	var subUserinfo string
	if s, ok := userinfoData["sub"].(string); ok {
		subUserinfo = s
	} else {
		return nil, nil, errors.Oauth2.Messagef("missing sub property in userinfo response, response=%q", userinfoResponseString)
	}

	if subIdtoken != subUserinfo {
		return nil, nil, errors.Oauth2.Messagef("subject mismatch, in ID token %q, in userinfo response %q", subIdtoken, subUserinfo)
	}

	return idTokenClaims, userinfoData, nil
}

func (a *AcClient) requestUserinfo(ctx context.Context, accessToken string) ([]byte, error) {
	userinfoReq, err := a.newUserinfoRequest(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userinfoRes, err := a.Backend.RoundTrip(userinfoReq)
	if err != nil {
		return nil, err
	}

	userinfoResBytes, err := ioutil.ReadAll(userinfoRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(a.asConfig.Reference()).Message("userinfo request read error").With(err)
	}

	if userinfoRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(a.asConfig.Reference()).Messagef("userinfo request failed, response=%q", string(userinfoResBytes))
	}

	return userinfoResBytes, nil
}

func (a *AcClient) newUserinfoRequest(ctx context.Context, accessToken string) (*http.Request, error) {
	userinfoEndpoint := a.getOidcAsConfig().GetUserinfoEndpoint()
	if userinfoEndpoint == "" {
		return nil, errors.Oauth2.Message("missing userinfo_endpoint in config")
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodGet, userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Authorization", "Bearer "+accessToken)

	outCtx := context.WithValue(ctx, request.URLAttribute, userinfoEndpoint)

	return outreq.WithContext(outCtx), nil
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
