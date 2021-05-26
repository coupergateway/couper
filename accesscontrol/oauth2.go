package accesscontrol

import (
	"context"
	"encoding/json"
	"net/http"

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
	if conf.GetGrantType() != grantType {
		return nil, confErr.Message("grant_type not supported: " + conf.GetGrantType())
	}

	return &OAuth2Callback{
		config: conf,
		oauth2: oauth2,
	}, nil
}

// Validate implements the AccessControl interface
func (oa *OAuth2Callback) Validate(req *http.Request) error {
	if req.Method != http.MethodGet {
		return errors.OAuth2.Messagef("wrong method: %s", req.Method)
	}

	code := req.URL.Query().Get("code")
	if code == "" {
		return errors.OAuth2.Messagef("missing code query parameter; query='%s'", req.URL.RawQuery)
	}

	requestConfig, err := oa.oauth2.GetRequestConfig(req)
	if err != nil {
		return errors.OAuth2.With(err)
	}

	requestConfig.Code = code

	tokenResponse, err := oa.oauth2.RequestToken(req.Context(), requestConfig)
	var jData map[string]interface{}

	err = json.Unmarshal([]byte(tokenResponse), &jData)
	if err != nil {
		return errors.OAuth2.With(err)
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
