package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
)

// OpenidConfiguration represents an OpenID configuration (.../.well-known/openid-configuration)
type OpenidConfiguration struct {
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	Issuer                        string   `json:"issuer"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserinfoEndpoint              string   `json:"userinfo_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

var _ config.OidcAS = &OidcConfig{}
var _ config.OAuth2Authorization = &OidcConfig{}
var _ config.OAuth2AcClient = &OidcConfig{}

// OidcConfig represents the configuration for an OIDC client
type OidcConfig struct {
	*config.OIDC
	Backend               http.RoundTripper
	memStore              *cache.MemoryStore
	ttl                   int64
	AuthorizationEndpoint string
	Issuer                string
	TokenEndpoint         string
	UserinfoEndpoint      string
}

// NewOidcConfig creates a new configuration for an OIDC client
func NewOidcConfig(oidc *config.OIDC, backend http.RoundTripper, memStore *cache.MemoryStore) (*OidcConfig, error) {
	ttl, parseErr := time.ParseDuration(oidc.TTL)
	if parseErr != nil {
		return nil, parseErr
	}
	return &OidcConfig{OIDC: oidc, Backend: backend, memStore: memStore, ttl: (int64)(ttl)}, nil
}

func (o *OidcConfig) Reference() string {
	return o.OIDC.BackendName
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or nonce)
func (o *OidcConfig) GetVerifierMethod() (string, error) {
	if o.VerifierMethod == "" {
		err := o.getFreshIfExpired()
		if err != nil {
			return "", err
		}
	}
	return o.VerifierMethod, nil
}

func (o *OidcConfig) GetAuthorizationEndpoint() (string, error) {
	err := o.getFreshIfExpired()
	if err != nil {
		return "", err
	}

	return o.AuthorizationEndpoint, nil
}

func (o *OidcConfig) GetIssuer() (string, error) {
	err := o.getFreshIfExpired()
	if err != nil {
		return "", err
	}

	return o.Issuer, nil
}

func (o *OidcConfig) GetTokenEndpoint() (string, error) {
	err := o.getFreshIfExpired()
	if err != nil {
		return "", err
	}

	return o.TokenEndpoint, nil
}

func (o *OidcConfig) GetUserinfoEndpoint() (string, error) {
	err := o.getFreshIfExpired()
	if err != nil {
		return "", err
	}

	return o.UserinfoEndpoint, nil
}

func (o *OidcConfig) getFreshIfExpired() error {
	stored := o.memStore.Get(o.ConfigurationURL)
	var (
		openidConfiguration *OpenidConfiguration
		err                 error
	)
	if stored != "" {
		openidConfiguration = &OpenidConfiguration{}
		decoder := json.NewDecoder(strings.NewReader(stored))
		err = decoder.Decode(openidConfiguration)
		if err != nil {
			return err
		}
	} else {
		openidConfiguration, err = o.fetchOpenidConfiguration()
		if err != nil {
			return err
		}
	}

	o.AuthorizationEndpoint = openidConfiguration.AuthorizationEndpoint
	o.Issuer = openidConfiguration.Issuer
	o.TokenEndpoint = openidConfiguration.TokenEndpoint
	o.UserinfoEndpoint = openidConfiguration.UserinfoEndpoint
	if o.OIDC.VerifierMethod == "" {
		if supportsS256(openidConfiguration.CodeChallengeMethodsSupported) {
			o.OIDC.VerifierMethod = config.CcmS256
		} else {
			o.OIDC.VerifierMethod = "nonce"
		}
	}

	return nil
}

func supportsS256(codeChallengeMethodsSupported []string) bool {
	if codeChallengeMethodsSupported == nil {
		return false
	}
	for _, codeChallengeMethod := range codeChallengeMethodsSupported {
		if codeChallengeMethod == "S256" {
			return true
		}
	}
	return false
}

func (o *OidcConfig) fetchOpenidConfiguration() (*OpenidConfiguration, error) {
	req, err := http.NewRequest(http.MethodGet, "", nil)
	ctx := context.WithValue(context.Background(), request.URLAttribute, o.ConfigurationURL)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}

	res, err := o.Backend.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	ocBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(ocBytes))
	openidConfiguration := &OpenidConfiguration{}
	err = decoder.Decode(openidConfiguration)
	if err != nil {
		return nil, err
	}

	o.memStore.Set(o.ConfigurationURL, string(ocBytes), o.ttl)
	return openidConfiguration, nil
}
