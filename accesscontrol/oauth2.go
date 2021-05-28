package accesscontrol

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler/transport"
)

var _ AccessControl = &OAuth2Callback{}

type OAuth2Callback struct {
	config *config.OAuth2AC
	oauth2 *transport.OAuth2
}

// NewOAuth2 creates a new AC-OAuth2 object
func NewOAuth2Callback(conf *config.OAuth2AC, oauth2 *transport.OAuth2) (*OAuth2Callback, error) {
	confErr := errors.Configuration.Label(conf.Name)

	const grantType = "authorization_code"
	if conf.GrantType != grantType {
		return nil, confErr.Messagef("grant_type %s not supported", conf.GrantType)
	}
	if conf.CsrfTokenParam != "" && conf.CsrfTokenParam != "state" {
		return nil, confErr.Messagef("csrf_token_param %s not supported", conf.CsrfTokenParam)
	}

	return &OAuth2Callback{
		config: conf,
		oauth2: oauth2,
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

	tokenResponse, err := oa.oauth2.RequestToken(req.Context(), requestConfig)
	if err != nil {
		return errors.Oauth2.Message("requesting token failed").With(err)
	}

	tokenResponseString := string(tokenResponse)
	var jData map[string]interface{}
	err = json.Unmarshal(tokenResponse, &jData)
	if err != nil {
		return errors.Oauth2.Messagef("parsing token response JSON failed, response='%s'", tokenResponseString).With(err)
	}

	if _, ok := jData["access_token"]; !ok {
		return errors.Oauth2.Messagef("missing access_token property in token response, response='%s'", tokenResponseString)
	}
	if _, ok := jData["expires_in"]; !ok {
		return errors.Oauth2.Messagef("missing expires_in property in token response, response='%s'", tokenResponseString)
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[oa.config.Name] = jData
	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
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
