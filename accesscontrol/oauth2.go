package accesscontrol

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/seetie"
)

var _ AccessControl = &OAuth2Callback{}

type OAuth2Callback struct {
	config    *config.OAuth2AC
	oauth2    *transport.OAuth2
	jwtParser *jwt.Parser
}

// NewOAuth2Callback creates a new AC-OAuth2 object
func NewOAuth2Callback(conf *config.OAuth2AC, oauth2 *transport.OAuth2) (*OAuth2Callback, error) {
	const grantType = "authorization_code"
	if conf.GrantType != grantType {
		return nil, fmt.Errorf("grant_type %s not supported", conf.GrantType)
	}
	if conf.Pkce == nil && conf.Csrf == nil {
		return nil, fmt.Errorf("CSRF protection not configured")
	}
	if conf.Csrf != nil {
		if conf.Csrf.TokenParam != "state" && conf.Csrf.TokenParam != "nonce" {
			return nil, fmt.Errorf("csrf_token_param %s not supported", conf.Csrf.TokenParam)
		}
		content, _, diags := conf.Csrf.HCLBody().PartialContent(conf.Csrf.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		conf.Csrf.Content = content
	}
	if conf.Pkce != nil {
		if conf.Pkce.CodeChallengeMethod != lib.CcmPlain && conf.Pkce.CodeChallengeMethod != lib.CcmS256 {
			return nil, fmt.Errorf("code_challenge_method %s not supported", conf.Pkce.CodeChallengeMethod)
		}
		content, _, diags := conf.Pkce.HCLBody().PartialContent(conf.Pkce.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		conf.Pkce.Content = content
	}

	options := []jwt.ParserOption{
		// jwt.WithValidMethods([]string{algo.String()}),
		jwt.WithLeeway(time.Second),
		// 2. The Issuer Identifier for the OpenID Provider (which is typically
		//    obtained during Discovery) MUST exactly match the value of the iss
		//    (issuer) Claim.
		jwt.WithIssuer(conf.Issuer),
		// 3. The Client MUST validate that the aud (audience) Claim contains its
		//    client_id value registered at the Issuer identified by the iss
		//    (issuer) Claim as an audience. The aud (audience) Claim MAY contain
		//    an array with more than one element. The ID Token MUST be rejected if
		//    the ID Token does not list the Client as a valid audience, or if it
		//    contains additional audiences not trusted by the Client.
		jwt.WithAudience(conf.ClientID),
	}
	jwtParser := jwt.NewParser(options...)

	return &OAuth2Callback{
		config:    conf,
		jwtParser: jwtParser,
		oauth2:    oauth2,
	}, nil
}

// Validate implements the AccessControl interface
func (oa *OAuth2Callback) Validate(req *http.Request) error {
	if req.Method != http.MethodGet {
		return errors.Oauth2.Messagef("wrong method: %s", req.Method)
	}

	query := req.URL.Query()
	code := query.Get("code")
	if code == "" {
		return errors.Oauth2.Messagef("missing code query parameter; query=%q", req.URL.RawQuery)
	}

	requestParams := map[string]string{"code": code, "redirect_uri": *oa.config.RedirectURI}

	ctx := req.Context()
	evalContext, _ := ctx.Value(eval.ContextType).(*eval.Context)

	if oa.config.Pkce != nil {
		v, _ := oa.config.Pkce.Content.Attributes["code_verifier_value"]
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		codeVerifierValue := strings.TrimSpace(seetie.ValueToString(ctyVal))
		if codeVerifierValue == "" {
			return errors.Oauth2.Message("Empty PKCE code_verifier_value")
		}
		requestParams["code_verifier"] = codeVerifierValue
	}

	var csrfToken, csrfTokenValue string
	if oa.config.Csrf != nil {
		v, _ := oa.config.Csrf.Content.Attributes["token_value"]
		ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
		csrfTokenValue = strings.TrimSpace(seetie.ValueToString(ctyVal))
		if csrfTokenValue == "" {
			return errors.Oauth2.Message("Empty CSRF token_value")
		}
		csrfToken = Base64urlSha256(csrfTokenValue)

		// validate state param value against CSRF token
		if oa.config.Csrf.TokenParam == "state" {
			csrfTokenFromParam := query.Get(oa.config.Csrf.TokenParam)
			if csrfTokenFromParam == "" {
				return errors.Oauth2.Messagef("missing state query parameter; query=%q", req.URL.RawQuery)
			}

			if csrfToken != csrfTokenFromParam {
				return errors.Oauth2.Messagef("CSRF token mismatch: %q (from query param) vs. %q (s256: %q)", csrfTokenFromParam, csrfTokenValue, csrfToken)
			}
		}
	}

	tokenResponse, err := oa.oauth2.RequestToken(ctx, requestParams)
	if err != nil {
		return errors.Oauth2.Message("requesting token failed").With(err)
	}

	tokenData, accessToken, err := transport.ParseAccessToken(tokenResponse)
	if err != nil {
		return errors.Oauth2.Messagef("parsing token response JSON failed, response=%q", string(tokenResponse)).With(err)
	}

	if idTokenString, ok := tokenData["id_token"].(string); ok {
		idToken, _, err := oa.jwtParser.ParseUnverified(idTokenString, jwt.MapClaims{})
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
		if err := idToken.Claims.Valid(oa.jwtParser.ValidationHelper); err != nil {
			return err
		}

		idtc, userinfo, err := oa.validateIdTokenClaims(ctx, idToken.Claims, csrfToken, csrfTokenValue, accessToken)
		if err != nil {
			return err
		}

		tokenData["id_token_claims"] = idtc
		tokenData["userinfo"] = userinfo
	}

	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[oa.config.Name] = tokenData
	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}

func (oa *OAuth2Callback) validateIdTokenClaims(ctx context.Context, claims jwt.Claims, csrfToken, csrfTokenValue string, accessToken string) (map[string]interface{}, map[string]interface{}, error) {
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
	if azpExists && azp != oa.config.ClientID {
		return nil, nil, errors.Oauth2.Messagef("azp claim / client ID mismatch, azp = %q, client ID = %q", azp, oa.config.ClientID)
	}

	// validate nonce claim value against CSRF token
	if oa.config.Csrf != nil && oa.config.Csrf.TokenParam == "nonce" {
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

	userinfoResponse, err := oa.requestUserinfo(ctx, accessToken)
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

func (oa *OAuth2Callback) requestUserinfo(ctx context.Context, accessToken string) ([]byte, error) {
	userinfoReq, err := oa.newUserinfoRequest(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userinfoRes, err := oa.oauth2.Backend.RoundTrip(userinfoReq)
	if err != nil {
		return nil, err
	}

	userinfoResBytes, err := ioutil.ReadAll(userinfoRes.Body)
	if err != nil {
		return nil, errors.Backend.Label(oa.config.Reference()).Message("userinfo request read error").With(err)
	}

	if userinfoRes.StatusCode != http.StatusOK {
		return nil, errors.Backend.Label(oa.config.Reference()).Messagef("userinfo request failed, response=%q", string(userinfoResBytes))
	}

	return userinfoResBytes, nil
}

func (oa *OAuth2Callback) newUserinfoRequest(ctx context.Context, accessToken string) (*http.Request, error) {
	if oa.config.UserinfoEndpoint == "" {
		return nil, errors.Oauth2.Message("missing userinfo_endpoint in config")
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodGet, oa.config.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Authorization", "Bearer "+accessToken)

	outCtx := context.WithValue(ctx, request.URLAttribute, oa.config.UserinfoEndpoint)

	return outreq.WithContext(outCtx), nil
}

func Base64urlSha256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
