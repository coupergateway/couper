package oauth2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/oauth2/oidc"
)

// OidcClient represents an open-id-connect client using the authorization code flow.
type OidcClient struct {
	*AuthCodeClient
	config    *oidc.Config
	issLock   sync.RWMutex
	issuer    string
	jwtParser *jwt.Parser
}

// NewOidcClient creates a new OIDC client.
func NewOidcClient(oidcConfig *oidc.Config) (*OidcClient, error) {
	acClient, err := NewAuthCodeClient(oidcConfig, oidcConfig, oidcConfig.AuthorizationBackend())
	if err != nil {
		return nil, err
	}

	o := &OidcClient{
		AuthCodeClient: acClient,
		config:         oidcConfig,
	}
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

func (o *OidcClient) getOidcAsConfig() config.OidcAS {
	oidcAsConfig, _ := o.asConfig.(config.OidcAS)
	return oidcAsConfig
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
		idToken, err := jwtParser.Parse(idTokenString, o.config.JWKS.GetSigKeyForToken)
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

		idtc, userinfo, err := o.validateIdTokenClaims(ctx, idToken.Claims, hashedVerifierValue, verifierValue, accessToken)
		if err != nil {
			return err
		}

		tokenResponseData["id_token_claims"] = idtc
		tokenResponseData["userinfo"] = userinfo

		return nil
	}

	return errors.Oauth2.Message("missing id_token in token response")
}

func (o *OidcClient) validateIdTokenClaims(ctx context.Context, claims jwt.Claims, hashedVerifierValue, verifierValue string, accessToken string) (map[string]interface{}, map[string]interface{}, error) {
	var idTokenClaims jwt.MapClaims
	if tc, ok := claims.(jwt.MapClaims); ok {
		idTokenClaims = tc
	}

	// 2.  ID Token
	// aud
	// 		REQUIRED.
	aud, audExists := idTokenClaims["aud"]
	if !audExists {
		return nil, nil, errors.Oauth2.Messagef("missing aud claim in ID token, claims='%#v'", idTokenClaims)
	}
	if aud == nil {
		return nil, nil, errors.Oauth2.Message("aud claim in ID token must not be null")
	}
	// exp
	// 		REQUIRED.
	if _, expExists := idTokenClaims["exp"]; !expExists {
		return nil, nil, errors.Oauth2.Messagef("missing exp claim in ID token, claims='%#v'", idTokenClaims)
	}
	// iat
	// 		REQUIRED.
	iat, iatExists := idTokenClaims["iat"]
	if !iatExists {
		return nil, nil, errors.Oauth2.Messagef("missing iat claim in ID token, claims='%#v'", idTokenClaims)
	}
	if _, ok := iat.(float64); !ok {
		return nil, nil, errors.Oauth2.Messagef("iat claim in ID token must be number, claims='%#v'", idTokenClaims)
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
	if azpExists && azp != o.clientConfig.GetClientID() {
		return nil, nil, errors.Oauth2.Messagef("azp claim / client ID mismatch, azp = %q, client ID = %q", azp, o.clientConfig.GetClientID())
	}

	verifierMethod, err := getVerifierMethod(ctx, o.asConfig)
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
			return nil, nil, errors.Oauth2.Messagef("missing nonce claim in ID token, claims='%#v'", idTokenClaims)
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

	userinfoResponse, err := o.requestUserinfo(ctx, accessToken)
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

func (o *OidcClient) requestUserinfo(ctx context.Context, accessToken string) ([]byte, error) {
	userinfoReq, err := o.newUserinfoRequest(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userinfoRes, err := o.Backend.RoundTrip(userinfoReq)
	if err != nil {
		return nil, err
	}

	userinfoResBytes, err := io.ReadAll(userinfoRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(o.asConfig.Reference()).Message("userinfo request read error").With(err)
	}

	if userinfoRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(o.asConfig.Reference()).Messagef("userinfo request failed, response=%q", string(userinfoResBytes))
	}

	return userinfoResBytes, nil
}

func (o *OidcClient) newUserinfoRequest(ctx context.Context, accessToken string) (*http.Request, error) {
	userinfoEndpoint, err := o.getOidcAsConfig().GetUserinfoEndpoint()
	if err != nil {
		return nil, err
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodGet, "", nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Authorization", "Bearer "+accessToken)

	outCtx := context.WithValue(ctx, request.URLAttribute, userinfoEndpoint)

	return outreq.WithContext(outCtx), nil
}
