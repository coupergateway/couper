package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/backend"
	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	jsn "github.com/avenga/couper/json"
)

// OpenidConfiguration represents an OpenID configuration (.../.well-known/openid-configuration)
type OpenidConfiguration struct {
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	Issuer                        string   `json:"issuer"`
	JwksUri                       string   `json:"jwks_uri"`
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
	backends   map[string]http.RoundTripper
	context    context.Context
	syncedJSON *jsn.SyncedJSON
	jwks       *jwk.JWKS
	cmu        sync.RWMutex // conf
	jmu        sync.RWMutex // jkws
}

// NewConfig creates a new configuration for an OIDC client
func NewConfig(oidc *config.OIDC, backends map[string]http.RoundTripper) (*Config, error) {
	ttl := defaultTTL
	if oidc.ConfigurationTTL != "" {
		t, err := time.ParseDuration(oidc.ConfigurationTTL)
		if err != nil {
			return nil, err
		}
		ttl = t
	}

	ctx := context.Background()
	conf := &Config{
		OIDC:     oidc,
		backends: backends,
		context:  ctx,
	}

	var err error
	conf.syncedJSON, err = jsn.NewSyncedJSON("", "",
		oidc.ConfigurationURL, backends["configuration_backend"], oidc.Name, ttl, conf)
	return conf, err
}

func (c *Config) Backends() map[string]http.RoundTripper {
	return c.backends
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or nonce)
func (c *Config) GetVerifierMethod(uid string) (string, error) {
	if c.VerifierMethod == "" {
		_, err := c.Data(uid)
		if err != nil {
			return "", err
		}
	}

	return c.VerifierMethod, nil
}

func (c *Config) GetAuthorizationEndpoint(uid string) (string, error) {
	openidConfigurationData, err := c.Data(uid)
	if err != nil {
		return "", err
	}

	return openidConfigurationData.AuthorizationEndpoint, nil
}

func (c *Config) GetIssuer() (string, error) {
	openidConfigurationData, err := c.Data("")
	if err != nil {
		return "", err
	}

	return openidConfigurationData.Issuer, nil
}

func (c *Config) GetTokenEndpoint() (string, error) {
	openidConfigurationData, err := c.Data("")
	if err != nil {
		return "", err
	}

	return openidConfigurationData.TokenEndpoint, nil
}

func (c *Config) GetUserinfoEndpoint() (string, error) {
	openidConfigurationData, err := c.Data("")
	if err != nil {
		return "", err
	}

	return openidConfigurationData.UserinfoEndpoint, nil
}

func (c *Config) Data(uid string) (*OpenidConfiguration, error) {
	data, err := c.syncedJSON.Data(uid)
	if err != nil {
		return nil, err
	}

	openidConfigurationData, ok := data.(*OpenidConfiguration)
	if !ok {
		return nil, fmt.Errorf("data not OpenID configuration data: %#v", data)
	}

	return openidConfigurationData, nil
}

func (c *Config) JWKS() *jwk.JWKS {
	c.jmu.RLock()
	defer c.jmu.RUnlock()
	j := *c.jwks
	return &j
}

func (c *Config) Unmarshal(rawJSON []byte) (interface{}, error) {
	jsonData := &OpenidConfiguration{}
	err := json.Unmarshal(rawJSON, jsonData)
	if err != nil {
		return nil, err
	}

	c.cmu.Lock()
	defer c.cmu.Unlock()

	if c.OIDC.VerifierMethod == "" {
		if supportsS256(jsonData.CodeChallengeMethodsSupported) {
			c.OIDC.VerifierMethod = config.CcmS256
		} else {
			c.OIDC.VerifierMethod = "nonce"
		}
	}

	jwksBackend := backend.NewContext(hclbody.
		New(hclbody.NewContentWithAttrName("_backend_url", jsonData.JwksUri)),
		c.backends["jwks_uri_backend"],
	)

	newJWKS, err := jwk.NewJWKS(jsonData.JwksUri, c.OIDC.ConfigurationTTL, jwksBackend)
	if err != nil { // do not replace possible working jwks on err
		return jsonData, err
	}

	c.jmu.Lock()
	c.jwks = newJWKS
	c.jmu.Unlock()

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
