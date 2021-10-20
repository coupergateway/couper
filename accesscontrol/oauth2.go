package accesscontrol

import (
	"context"
	"net/http"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/oauth2"
)

var _ AccessControl = &OAuth2Callback{}

// OAuth2Callback represents the access control for the OAuth2 authorization code flow callback.
type OAuth2Callback struct {
	oauth2Client oauth2.AcClient
}

// NewOAuth2Callback creates a new access control for the OAuth2 authorization code flow callback.
func NewOAuth2Callback(oauth2Client oauth2.AcClient) (*OAuth2Callback, error) {
	return &OAuth2Callback{
		oauth2Client: oauth2Client,
	}, nil
}

// Validate implements the AccessControl interface
func (oa *OAuth2Callback) Validate(req *http.Request) error {
	if req.Method != http.MethodGet {
		return errors.Oauth2.Messagef("wrong method (%s)", req.Method)
	}

	tokenResponseData, err := oa.oauth2Client.GetTokenResponse(req, req.URL)
	if err != nil {
		return err
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[oa.oauth2Client.GetName()] = tokenResponseData
	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}
