package accesscontrol

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
)

var _ AccessControl = &OAuth2Callback{}

type OAuth2Callback struct {
	config    *config.OAuth2AC
	oauth2    *transport.OAuth2
	jwtParser *jwt.Parser
}

// NewOAuth2Callback creates a new AC-OAuth2 object
func NewOAuth2Callback(conf *config.OAuth2AC, oauth2 *transport.OAuth2) (*OAuth2Callback, error) {
	confErr := errors.Configuration.Label(conf.Name)

	const grantType = "authorization_code"
	if conf.GrantType != grantType {
		return nil, confErr.Messagef("grant_type %s not supported", conf.GrantType)
	}
	if conf.CodeChallengeMethod == "" && conf.CsrfTokenParam == "" {
		return nil, confErr.Message("CSRF protection not configured")
	}
	if conf.CsrfTokenParam != "" && conf.CsrfTokenParam != "state" && conf.CsrfTokenParam != "nonce" {
		return nil, confErr.Messagef("csrf_token_param %s not supported", conf.CsrfTokenParam)
	}
	if conf.CodeChallengeMethod != "" && conf.CodeChallengeMethod != lib.CCM_plain && conf.CodeChallengeMethod != lib.CCM_S256 {
		return nil, confErr.Messagef("code_challenge_method %s not supported", conf.CodeChallengeMethod)
	}

	options := []jwt.ParserOption{
		// jwt.WithValidMethods([]string{algo.String()}),
		jwt.WithLeeway(time.Second),
	}
	// 2. The Issuer Identifier for the OpenID Provider (which is typically
	//    obtained during Discovery) MUST exactly match the value of the iss
	//    (issuer) Claim.
	options = append(options, jwt.WithIssuer(conf.Issuer))
	// 3. The Client MUST validate that the aud (audience) Claim contains its
	//    client_id value registered at the Issuer identified by the iss
	//    (issuer) Claim as an audience. The aud (audience) Claim MAY contain
	//    an array with more than one element. The ID Token MUST be rejected if
	//    the ID Token does not list the Client as a valid audience, or if it
	//    contains additional audiences not trusted by the Client.
	options = append(options, jwt.WithAudience(conf.ClientID))
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
		return errors.Oauth2.Messagef("missing code query parameter; query='%s'", req.URL.RawQuery)
	}

	requestConfig, err := oa.oauth2.GetRequestConfig(req)
	if err != nil {
		return errors.Oauth2.With(err)
	}

	// validate state param value against CSRF token
	if oa.config.CsrfTokenParam == "state" {
		csrfTokenFromParam := query.Get(oa.config.CsrfTokenParam)
		if csrfTokenFromParam == "" {
			return errors.Oauth2.Messagef("missing state query parameter; query='%s'", req.URL.RawQuery)
		}
		csrfToken := Base64url_s256(*requestConfig.CsrfToken)
		if csrfToken != csrfTokenFromParam {
			return errors.Oauth2.Messagef("CSRF token mismatch: '%s' (from query param) vs. '%s' (s256: '%s')", csrfTokenFromParam, *requestConfig.CsrfToken, csrfToken)
		}
	}

	requestConfig.Code = &code
	requestConfig.RedirectURI = oa.config.RedirectURI

	tokenResponse, err := oa.oauth2.RequestToken(req.Context(), requestConfig)
	if err != nil {
		return errors.Oauth2.Message("requesting token failed").With(err)
	}

	tokenData, accessToken, err := transport.ParseAccessToken(tokenResponse)
	if err != nil {
		return errors.Oauth2.Messagef("parsing token response JSON failed, response='%s'", string(tokenResponse)).With(err)
	}

	ctx := req.Context()
	if idTokenString, ok := tokenData["id_token"].(string); ok {
		// TODO use ParseWithClaims() with key function instead
		idToken, _, err := oa.jwtParser.ParseUnverified(idTokenString, jwt.MapClaims{})
		if err != nil {
			return err
		}

		idtc, err := oa.validateIdTokenClaims(idToken.Claims, requestConfig, ctx, accessToken)
		if err != nil {
			return err
		}

		tokenData["id_token"] = idtc
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

func (oa *OAuth2Callback) validateIdTokenClaims(claims jwt.Claims, requestConfig *transport.OAuth2RequestConfig, ctx context.Context, accessToken string) (map[string]interface{}, error) {
	var idTokenClaims jwt.MapClaims
	if tc, ok := claims.(jwt.MapClaims); ok {
		idTokenClaims = tc
	}

	// 4. If the ID Token contains multiple audiences, the Client SHOULD verify
	//    that an azp Claim is present.
	azp, azpExists := idTokenClaims["azp"]
	if auds, audsOK := idTokenClaims["aud"].([]interface{}); audsOK && len(auds) > 1 && !azpExists {
		return nil, errors.Oauth2.Messagef("missing azp claim in ID token, claims='%#v'", idTokenClaims)
	}
	// 5. If an azp (authorized party) Claim is present, the Client SHOULD
	//    verify that its client_id is the Claim Value.
	if azpExists && azp != oa.config.ClientID {
		return nil, errors.Oauth2.Messagef("azp claim / client ID mismatch, azp = '%s', client ID = '%s'", azp, oa.config.ClientID)
	}

	// validate nonce claim value against CSRF token
	if oa.config.CsrfTokenParam == "nonce" {
		// 11. If a nonce value was sent in the Authentication Request, a nonce
		//     Claim MUST be present and its value checked to verify that it is the
		//     same value as the one that was sent in the Authentication Request.
		//     The Client SHOULD check the nonce value for replay attacks. The
		//     precise method for detecting replay attacks is Client specific.
		var nonce string
		if n, ok := idTokenClaims["nonce"].(string); ok {
			nonce = n
		} else {
			return nil, errors.Oauth2.Messagef("missing nonce claim in ID token, claims='%#v'", idTokenClaims)
		}

		csrfToken := Base64url_s256(*requestConfig.CsrfToken)
		if csrfToken != nonce {
			return nil, errors.Oauth2.Messagef("CSRF token mismatch: '%s' (from nonce claim) vs. '%s' (s256: '%s')", nonce, *requestConfig.CsrfToken, csrfToken)
		}
	}

	var subIdtoken string
	if s, ok := idTokenClaims["sub"].(string); ok {
		subIdtoken = s
	} else {
		return nil, errors.Oauth2.Messagef("missing sub claim in ID token, claims='%#v'", idTokenClaims)
	}

	userinfoResponse, err := oa.requestUserinfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userinfoResponseString := string(userinfoResponse)
	var userinfoData map[string]interface{}
	err = json.Unmarshal(userinfoResponse, &userinfoData)
	if err != nil {
		return nil, errors.Oauth2.Messagef("parsing userinfo response JSON failed, response='%s'", userinfoResponseString).With(err)
	}

	var subUserinfo string
	if s, ok := userinfoData["sub"].(string); ok {
		subUserinfo = s
	} else {
		return nil, errors.Oauth2.Messagef("missing sub property in userinfo response, response='%s'", userinfoResponseString)
	}

	if subIdtoken != subUserinfo {
		return nil, errors.Oauth2.Messagef("subject mismatch, in ID token: '%s', in userinfo response: '%s'", subIdtoken, subUserinfo)
	}

	return idTokenClaims, nil
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
		return nil, errors.Backend.Label(oa.config.Reference()).Messagef("userinfo request failed, response='%s'", string(userinfoResBytes))
	}

	return userinfoResBytes, nil
}

func (oa *OAuth2Callback) newUserinfoRequest(ctx context.Context, accessToken string) (*http.Request, error) {
	url, err := eval.GetContextAttribute(oa.config.HCLBody(), ctx, "userinfo_endpoint")
	if err != nil {
		return nil, err
	}

	if url == "" {
		return nil, errors.Oauth2.Messagef("missing userinfo_endpoint in config")
	}

	// url will be configured via backend roundtrip
	outreq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	outreq.Header.Set("Authorization", "Bearer "+accessToken)

	outCtx := context.WithValue(ctx, request.URLAttribute, url)

	return outreq.WithContext(outCtx), nil
}

func Base64url_s256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64url_encode(h.Sum(nil))
}

func base64url_encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}
