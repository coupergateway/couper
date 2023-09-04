package accesscontrol

import (
	"context"
	"net/http"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/oauth2"
)

var _ AccessControl = &OAuth2Callback{}

// OAuth2Callback represents the access control for the OAuth2 authorization code flow callback.
type OAuth2Callback struct {
	oauth2Client oauth2.AuthCodeFlowClient
	name         string
}

// NewOAuth2Callback creates a new access control for the OAuth2 authorization code flow callback.
func NewOAuth2Callback(oauth2Client oauth2.AuthCodeFlowClient, name string) *OAuth2Callback {
	return &OAuth2Callback{
		oauth2Client: oauth2Client,
		name:         name,
	}
}

// Validate implements the AccessControl interface
func (oa *OAuth2Callback) Validate(req *http.Request) error {
	if req.Method != http.MethodGet {
		return errors.Oauth2.Messagef("wrong method (%s)", req.Method)
	}

	tokenResponseData, err := oa.oauth2Client.ExchangeCodeAndGetTokenResponse(req, req.URL)
	if err != nil {
		return err
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[oa.name] = tokenResponseData
	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}
