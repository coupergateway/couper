package accesscontrol

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	jwks      *JWKS
}

// NewOAuth2Callback creates a new AC-OAuth2 object
func NewOAuth2Callback(conf *config.OAuth2AC, oauth2 *transport.OAuth2) (*OAuth2Callback, error) {
	confErr := errors.Configuration.Label(conf.Name)

	const grantType = "authorization_code"
	if conf.GrantType != grantType {
		return nil, confErr.Messagef("grant_type %s not supported", conf.GrantType)
	}
	if conf.Pkce == nil && conf.Csrf == nil {
		return nil, confErr.Message("CSRF protection not configured")
	}
	if conf.Csrf != nil && conf.Csrf.TokenParam != "state" && conf.Csrf.TokenParam != "nonce" {
		return nil, confErr.Messagef("csrf_token_param %s not supported", conf.Csrf.TokenParam)
	}
	if conf.Pkce != nil && conf.Pkce.CodeChallengeMethod != lib.CcmPlain && conf.Pkce.CodeChallengeMethod != lib.CcmS256 {
		return nil, confErr.Messagef("code_challenge_method %s not supported", conf.Pkce.CodeChallengeMethod)
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

	oa := &OAuth2Callback{
		config:    conf,
		jwtParser: jwtParser,
		oauth2:    oauth2,
	}

	if conf.JwksFile != "" {
		jwksReader, err := readJwksFile(conf.JwksFile)
		if err != nil {
			return nil, confErr.Messagef("error reading JWKS file").With(err)
		}

		jsonWebKeySet := new(JWKS)
		if err = json.NewDecoder(jwksReader).Decode(jsonWebKeySet); err != nil {
			return nil, confErr.Messagef("error parsing JWKS file").With(err)
		}
		oa.jwks = jsonWebKeySet
	}

	return oa, nil
}

func readJwksFile(filePath string) (io.Reader, error) {
	p, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	return os.Open(p)
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

	evalContext, _ := req.Context().Value(eval.ContextType).(*eval.Context)

	if oa.config.Pkce != nil {
		content, _, diags := oa.config.Pkce.HCLBody().PartialContent(oa.config.Pkce.Schema(true))
		if diags.HasErrors() {
			return errors.Evaluation.With(diags)
		}

		if v, ok := content.Attributes["code_verifier_value"]; ok {
			ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
			strVal := strings.TrimSpace(seetie.ValueToString(ctyVal))
			requestConfig.CodeVerifier = &strVal
		}
	}

	var csrfToken, csrfTokenValue string
	if oa.config.Csrf != nil {
		content, _, diags := oa.config.Csrf.HCLBody().PartialContent(oa.config.Csrf.Schema(true))
		if diags.HasErrors() {
			return errors.Evaluation.With(diags)
		}

		if v, ok := content.Attributes["token_value"]; ok {
			ctyVal, _ := v.Expr.Value(evalContext.HCLContext())
			csrfTokenValue = strings.TrimSpace(seetie.ValueToString(ctyVal))
		} else {
			return errors.Oauth2.Messagef("Missing CSRF token_value")
		}
		csrfToken = Base64url_s256(csrfTokenValue)

		// validate state param value against CSRF token
		if oa.config.Csrf.TokenParam == "state" {
			csrfTokenFromParam := query.Get(oa.config.Csrf.TokenParam)
			if csrfTokenFromParam == "" {
				return errors.Oauth2.Messagef("missing state query parameter; query='%s'", req.URL.RawQuery)
			}

			if csrfToken != csrfTokenFromParam {
				return errors.Oauth2.Messagef("CSRF token mismatch: '%s' (from query param) vs. '%s' (s256: '%s')", csrfTokenFromParam, csrfTokenValue, csrfToken)
			}
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
		if oa.jwks == nil {
			return errors.Oauth2.Messagef("missing JWKS")
		}

		idToken, err := oa.jwtParser.ParseWithClaims(idTokenString, jwt.MapClaims{}, oa.getKey)
		if err != nil {
			return err
		}

		idtc, err := oa.validateIdTokenClaims(idToken.Claims, csrfToken, csrfTokenValue, ctx, accessToken)
		if err != nil {
			return err
		}

		tokenData["id_token_claims"] = idtc
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

func (oa *OAuth2Callback) getKey(token *jwt.Token) (interface{}, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.Oauth2.Messagef("missing kid header in ID token, header='%#v'", token.Header)
	}

	alg, ok := token.Header["alg"].(string)
	if !ok {
		return nil, errors.Oauth2.Messagef("missing alg header in ID token, header='%#v'", token.Header)
	}

	keys := oa.jwks.Key(kid)
	if len(keys) == 0 {
		return nil, errors.Oauth2.Messagef("no key for kid '%s' found in JWKS", kid)
	}

	jwk := keys[0]
	if jwk.Algorithm != alg {
		return nil, errors.Oauth2.Messagef("alg mismatch for key with kid '%s', token alg = %s, jwk alg = %s", kid, alg, jwk.Algorithm)
	}

	if jwk.Use != "sig" {
		return nil, errors.Oauth2.Messagef("key with kid '%s' is not usable for signing", kid)
	}

	return jwk.Key, nil
}

func (oa *OAuth2Callback) validateIdTokenClaims(claims jwt.Claims, csrfToken, csrfTokenValue string, ctx context.Context, accessToken string) (map[string]interface{}, error) {
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
			return nil, errors.Oauth2.Messagef("missing nonce claim in ID token, claims='%#v'", idTokenClaims)
		}

		if csrfToken != nonce {
			return nil, errors.Oauth2.Messagef("CSRF token mismatch: '%s' (from nonce claim) vs. '%s' (s256: '%s')", nonce, csrfTokenValue, csrfToken)
		}
	}

	return idTokenClaims, nil
}

func Base64url_s256(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
