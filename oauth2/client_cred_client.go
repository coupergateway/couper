package oauth2

import (
	"net/http"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

// NewClientCredentialsClient creates a new OAuth2 Client Credentials client.
func NewClientCredentialsClient(conf *config.OAuth2ReqAuth, backend http.RoundTripper) (*Client, error) {
	backendErr := errors.Backend.Label(conf.Reference())
	// grant_type password undocumented feature!
	if conf.GrantType != "client_credentials" && conf.GrantType != "password" {
		return nil, backendErr.Messagef("grant_type %s not supported", conf.GrantType)
	}

	if conf.GrantType == "client_credentials" {
		// conf.Username undocumented feature!
		if conf.Username != "" {
			return nil, backendErr.Message("username must not be set with grant_type=client_credentials")
		}
		// conf.Password undocumented feature!
		if conf.Password != "" {
			return nil, backendErr.Message("password must not be set with grant_type=client_credentials")
		}
	}

	// grant_type password undocumented feature!
	// WARNING: this implementation is no proper password flow, but a flow with username and password to login _exactly one_ user
	// the received access token is stored in cache just like with the client credentials flow
	if conf.GrantType == "password" {
		if conf.Username == "" {
			return nil, backendErr.Message("username must not be empty with grant_type=password")
		}
		if conf.Password == "" {
			return nil, backendErr.Message("password must not be empty with grant_type=password")
		}
	}

	if teAuthMethod := conf.TokenEndpointAuthMethod; teAuthMethod != nil {
		if *teAuthMethod != "client_secret_basic" && *teAuthMethod != "client_secret_post" {
			return nil, backendErr.Messagef("token_endpoint_auth_method %s not supported", *teAuthMethod)
		}
	}
	return &Client{backend, conf, conf, conf.GrantType}, nil
}
