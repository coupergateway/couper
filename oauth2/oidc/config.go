package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/avenga/couper/config"
	jsn "github.com/avenga/couper/json"
)

// OpenidConfiguration represents an OpenID configuration (.../.well-known/openid-configuration)
type OpenidConfiguration struct {
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	Issuer                        string   `json:"issuer"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserinfoEndpoint              string   `json:"userinfo_endpoint"`
}

type Configs map[string]*Config

var (
	_ config.OidcAS              = &Config{}
	_ config.OAuth2Authorization = &Config{}
	_ config.OAuth2AcClient      = &Config{}

	defaultTTL = time.Hour
)

// Config represents the configuration for an OIDC client
type Config struct {
	*config.OIDC
	Backend    http.RoundTripper
	syncedJSON *jsn.SyncedJSON
}

// NewConfig creates a new configuration for an OIDC client
func NewConfig(oidc *config.OIDC, backend http.RoundTripper) (*Config, error) {
	ttl := defaultTTL
	if oidc.ConfigurationTTL != "" {
		t, err := time.ParseDuration(oidc.ConfigurationTTL)
		if err != nil {
			return nil, err
		}
		ttl = t
	}

	config := &Config{OIDC: oidc, Backend: backend}
	sj := jsn.NewSyncedJSON(context.Background(), "", "", oidc.ConfigurationURL, backend, oidc.Name, ttl, config)
	config.syncedJSON = sj
	return config, nil
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or nonce)
func (c *Config) GetVerifierMethod(uid string) (string, error) {
	if c.VerifierMethod == "" {
		_, err := c.Data()
		if err != nil {
			return "", err
		}
	}

	return c.VerifierMethod, nil
}

func (c *Config) GetAuthorizationEndpoint(uid string) (string, error) {
	openidConfigurationData, err := c.Data()
	if err != nil {
		return "", err
	}

	return openidConfigurationData.AuthorizationEndpoint, nil
}

func (c *Config) GetIssuer() (string, error) {
	openidConfigurationData, err := c.Data()
	if err != nil {
		return "", err
	}

	return openidConfigurationData.Issuer, nil
}

func (c *Config) GetTokenEndpoint() (string, error) {
	openidConfigurationData, err := c.Data()
	if err != nil {
		return "", err
	}

	return openidConfigurationData.TokenEndpoint, nil
}

func (c *Config) GetUserinfoEndpoint() (string, error) {
	openidConfigurationData, err := c.Data()
	if err != nil {
		return "", err
	}

	return openidConfigurationData.UserinfoEndpoint, nil
}

func (c *Config) Data() (*OpenidConfiguration, error) {
	data, err := c.syncedJSON.Data()
	if err != nil {
		return nil, err
	}

	openidConfigurationData, ok := data.(OpenidConfiguration)
	if !ok {
		return nil, fmt.Errorf("data not OpenID configuration data: %#v", data)
	}

	return &openidConfigurationData, nil
}

func (c *Config) Unmarshal(rawJSON []byte) (interface{}, error) {
	var jsonData OpenidConfiguration
	err := json.Unmarshal(rawJSON, &jsonData)
	if err != nil {
		return nil, err
	}

	if c.OIDC.VerifierMethod == "" {
		if supportsS256(jsonData.CodeChallengeMethodsSupported) {
			c.OIDC.VerifierMethod = config.CcmS256
		} else {
			c.OIDC.VerifierMethod = "nonce"
		}
	}
	return jsonData, nil
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
