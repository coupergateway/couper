package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/oauth2/oidc"
)

// OidcClient represents an OpenID Connect client using the authorization code flow.
type OidcClient struct {
	*AuthCodeClient
	config    *oidc.Config
	backends  map[string]http.RoundTripper
	issLock   sync.RWMutex
	issuer    string
	jwtParser *jwt.Parser
}

// NewOidcClient creates a new OIDC client.
func NewOidcClient(oidcConfig *oidc.Config) (*OidcClient, error) {
	o := &OidcClient{
		config:   oidcConfig,
		backends: oidcConfig.Backends(),
	}

	acClient, err := NewAuthCodeClient(oidcConfig, oidcConfig, o.backends["token_backend"])
	if err != nil {
		return nil, err
	}

	o.AuthCodeClient = acClient
	o.AuthCodeFlowClient = o
	return o, nil
}

func (o *OidcClient) refreshJWTParser() error {
	o.issLock.RLock()
	iss := o.issuer
	o.issLock.RUnlock()

	confIssuer, err := o.config.GetIssuer()
	if err != nil {
		return err
	}

	if iss == confIssuer {
		return nil
	}

	options := []jwt.ParserOption{
		// jwt.WithValidMethods([]string{algo.String()}),
		jwt.WithLeeway(time.Second),
		// 2. The Issuer Identifier for the OpenID Provider (which is typically
		//    obtained during Discovery) MUST exactly match the value of the iss
		//    (issuer) Claim.
		jwt.WithIssuer(confIssuer),
		// 3. The Client MUST validate that the aud (audience) Claim contains its
		//    client_id value registered at the Issuer identified by the iss
		//    (issuer) Claim as an audience. The aud (audience) Claim MAY contain
		//    an array with more than one element. The ID Token MUST be rejected if
		//    the ID Token does not list the Client as a valid audience, or if it
		//    contains additional audiences not trusted by the Client.
		jwt.WithAudience(o.config.GetClientID()),
	}
	jwtParser := jwt.NewParser(options...)

	o.issLock.Lock()
	o.issuer = confIssuer
	o.jwtParser = jwtParser
	o.issLock.Unlock()

	return nil
}

// validateTokenResponseData validates the token response data
func (o *OidcClient) validateTokenResponseData(ctx context.Context, tokenResponseData map[string]interface{}, hashedVerifierValue, verifierValue, accessToken string) error {
	if err := o.refreshJWTParser(); err != nil {
		return err
	}

	o.issLock.RLock()
	jwtParser := o.jwtParser
	o.issLock.RUnlock()

	if idTokenString, ok := tokenResponseData["id_token"].(string); ok {

		idToken, err := jwtParser.Parse(idTokenString, o.Keyfunc)
		if err != nil {
			return err
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
		if err = idToken.Claims.Valid(jwtParser.ValidationHelper); err != nil {
			return err
		}

		idtc, userinfo, err := o.validateIDTokenClaims(ctx, idToken.Claims, hashedVerifierValue, verifierValue, accessToken)
		if err != nil {
			return err
		}

		tokenResponseData["id_token_claims"] = idtc
		tokenResponseData["userinfo"] = userinfo

		return nil
	}

	return errors.Oauth2.Message("missing id_token in token response")
}

func (o *OidcClient) Keyfunc(token *jwt.Token) (interface{}, error) {
	return o.config.JWKS().
		GetSigKeyForToken(token)
}

func (o *OidcClient) validateIDTokenClaims(ctx context.Context, claims jwt.Claims, hashedVerifierValue, verifierValue string, accessToken string) (map[string]interface{}, map[string]interface{}, error) {
	var idTokenClaims jwt.MapClaims
	if tc, ok := claims.(jwt.MapClaims); ok {
		idTokenClaims = tc
	}

	// 2.  ID Token
	// aud
	// 		REQUIRED.
	aud, audExists := idTokenClaims["aud"]
	if !audExists {
		return nil, nil, errors.Oauth2.Message("missing aud claim in ID token")
	}
	if aud == nil {
		return nil, nil, errors.Oauth2.Message("aud claim in ID token must not be null")
	}
	// exp
	// 		REQUIRED.
	if _, expExists := idTokenClaims["exp"]; !expExists {
		return nil, nil, errors.Oauth2.Message("missing exp claim in ID token")
	}
	// iat
	// 		REQUIRED.
	iat, iatExists := idTokenClaims["iat"]
	if !iatExists {
		return nil, nil, errors.Oauth2.Message("missing iat claim in ID token")
	}
	if _, ok := iat.(float64); !ok {
		return nil, nil, errors.Oauth2.Messagef("iat claim in ID token must be number, but is %#v", iat)
	}

	// 3.1.3.7.  ID Token Validation
	// 4. If the ID Token contains multiple audiences, the Client SHOULD verify
	//    that an azp Claim is present.
	azp, azpExists := idTokenClaims["azp"]
	if auds, audsOK := idTokenClaims["aud"].([]interface{}); audsOK && len(auds) > 1 && !azpExists {
		return nil, nil, errors.Oauth2.Message("missing azp claim in ID token")
	}
	// 5. If an azp (authorized party) Claim is present, the Client SHOULD
	//    verify that its client_id is the Claim Value.
	if azpExists && azp != o.clientConfig.GetClientID() {
		return nil, nil, errors.Oauth2.Messagef("azp claim / client ID mismatch, azp = %q, client ID = %q", azp, o.clientConfig.GetClientID())
	}

	verifierMethod, err := o.config.GetVerifierMethod()
	if err != nil {
		return nil, nil, err
	}

	// validate nonce claim value against CSRF token
	if verifierMethod == "nonce" {
		// 11. If a nonce value was sent in the Authentication Request, a nonce
		//     Claim MUST be present and its value checked to verify that it is the
		//     same value as the one that was sent in the Authentication Request.
		//     The Client SHOULD check the nonce value for replay attacks. The
		//     precise method for detecting replay attacks is Client specific.
		var nonce string
		if n, ok := idTokenClaims["nonce"].(string); ok {
			nonce = n
		} else {
			return nil, nil, errors.Oauth2.Message("missing nonce claim in ID token")
		}

		if hashedVerifierValue != nonce {
			return nil, nil, errors.Oauth2.Messagef("nonce mismatch: %q (from nonce claim) vs. %q (verifier_value: %q)", nonce, hashedVerifierValue, verifierValue)
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

	userinfoData, subUserinfo, err := o.getUserinfo(ctx, accessToken)
	if err != nil {
		return nil, nil, errors.Oauth2.Message("userinfo request error").With(err)
	}

	if subIdtoken != subUserinfo {
		return nil, nil, errors.Oauth2.Messagef("subject mismatch, in ID token %q, in userinfo response %q", subIdtoken, subUserinfo)
	}

	return idTokenClaims, userinfoData, nil
}

func (o *OidcClient) getUserinfo(ctx context.Context, accessToken string) (map[string]interface{}, string, error) {
	userinfoReq, err := o.newUserinfoRequest(ctx, accessToken)
	if err != nil {
		return nil, "", err
	}

	userinfoResponse, err := o.requestUserinfo(userinfoReq)
	if err != nil {
		return nil, "", err
	}

	return parseUserinfoResponse(userinfoResponse)
}

func (o *OidcClient) requestUserinfo(userinfoReq *http.Request) ([]byte, error) {
	ctx, cancel := context.WithCancel(userinfoReq.Context())
	defer cancel()

	userinfoRes, err := o.backends["userinfo_backend"].RoundTrip(userinfoReq.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer userinfoRes.Body.Close()

	userinfoResBytes, err := io.ReadAll(userinfoRes.Body)
	if err != nil {
		return nil, err
	}

	if userinfoRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong status code, status=%d, response=%q", userinfoRes.StatusCode, string(userinfoResBytes))
	}

	return userinfoResBytes, nil
}

func parseUserinfoResponse(userinfoResponse []byte) (map[string]interface{}, string, error) {
	var userinfoData map[string]interface{}
	err := json.Unmarshal(userinfoResponse, &userinfoData)
	if err != nil {
		return nil, "", err
	}

	sub, ok := userinfoData["sub"].(string)
	if !ok {
		return nil, "", fmt.Errorf("missing sub property, response=%q", string(userinfoResponse))
	}

	return userinfoData, sub, nil
}

func (o *OidcClient) newUserinfoRequest(ctx context.Context, accessToken string) (*http.Request, error) {
	userinfoEndpoint, err := o.config.GetUserinfoEndpoint()
	if err != nil {
		return nil, err
	}

	outreq, err := http.NewRequest(http.MethodGet, userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Authorization", "Bearer "+accessToken)

	return outreq.WithContext(ctx), nil
}
