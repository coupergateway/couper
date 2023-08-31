package oidc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coupergateway/couper/accesscontrol/jwk"
	"github.com/coupergateway/couper/backend"
	"github.com/coupergateway/couper/config"
	hclbody "github.com/coupergateway/couper/config/body"
	jsn "github.com/coupergateway/couper/json"
)

// OpenidConfiguration represents an OpenID configuration (.../.well-known/openid-configuration)
type OpenidConfiguration struct {
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	Issuer                        string   `json:"issuer"`
	JwksURI                       string   `json:"jwks_uri"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserinfoEndpoint              string   `json:"userinfo_endpoint"`
}

type Configs map[string]*Config

var (
	_ config.OAuth2AS            = &Config{}
	_ config.OAuth2Authorization = &Config{}
	_ config.OAuth2AcClient      = &Config{}

	defaultTTL = time.Hour
)

// Config represents the configuration for an OIDC client
type Config struct {
	*config.OIDC
	backends     map[string]http.RoundTripper
	context      context.Context
	syncedJSON   *jsn.SyncedJSON
	jwks         *jwk.JWKS
	jwksCheckSum [32]byte
	jwksCancel   func()
	jmu          sync.RWMutex // jkws
}

// NewConfig creates a new configuration for an OIDC client
func NewConfig(ctx context.Context, oidc *config.OIDC, backends map[string]http.RoundTripper) (*Config, error) {
	ttl, err := config.ParseDuration("configuration_ttl", oidc.ConfigurationTTL, defaultTTL)
	if err != nil {
		return nil, err
	}
	maxStale, err := config.ParseDuration("configuration_max_stale", oidc.ConfigurationMaxStale, defaultTTL)
	if err != nil {
		return nil, err
	}

	conf := &Config{
		OIDC:     oidc,
		backends: backends,
		context:  ctx,
	}

	conf.syncedJSON, err = jsn.NewSyncedJSON(ctx, "", "",
		oidc.ConfigurationURL, backends["configuration_backend"], oidc.Name, ttl, maxStale, conf)
	if err != nil {
		return nil, err
	}

	// verify verifierMethod with locked access
	conf.jmu.RLock()
	defer conf.jmu.RUnlock()
	if conf.VerifierMethod != "" &&
		conf.VerifierMethod != config.CcmS256 &&
		conf.VerifierMethod != "nonce" {
		return nil, fmt.Errorf("verifier_method %s not supported", conf.VerifierMethod)
	}

	return conf, err
}

func (c *Config) Backends() map[string]http.RoundTripper {
	return c.backends
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or nonce)
func (c *Config) GetVerifierMethod() (string, error) {
	c.jmu.RLock()
	if c.VerifierMethod == "" {
		c.jmu.RUnlock()
		_, err := c.Data()
		if err != nil {
			return "", err
		}
		c.jmu.RLock()
	}
	defer c.jmu.RUnlock()

	return c.VerifierMethod, nil
}

func (c *Config) GetAuthorizationEndpoint() (string, error) {
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

	openidConfigurationData, ok := data.(*OpenidConfiguration)
	if !ok {
		return nil, fmt.Errorf("data not OpenID configuration data: %#v", data)
	}

	return openidConfigurationData, nil
}

func (c *Config) JWKS() *jwk.JWKS {
	c.jmu.RLock()
	defer c.jmu.RUnlock()
	return c.jwks
}

func (c *Config) Unmarshal(rawJSON []byte) (interface{}, error) {
	c.jmu.Lock()
	defer c.jmu.Unlock()

	jsonData := &OpenidConfiguration{}
	err := json.Unmarshal(rawJSON, jsonData)
	if err != nil {
		return nil, err
	}

	checkSum := sha256.Sum256(rawJSON)
	if bytes.Equal(checkSum[:], c.jwksCheckSum[:]) {
		// return obtained (same) data here since Data() call will block
		return jsonData, nil
	}
	c.jwksCheckSum = checkSum

	if c.OIDC.VerifierMethod == "" {
		if supportsS256(jsonData.CodeChallengeMethodsSupported) {
			c.OIDC.VerifierMethod = config.CcmS256
		} else {
			c.OIDC.VerifierMethod = "nonce"
		}
	}

	jwksBackend := backend.NewContext(
		hclbody.NewHCLSyntaxBodyWithStringAttr("_backend_url", jsonData.JwksURI),
		c.backends["jwks_uri_backend"],
	)

	ctx, cancel := context.WithCancel(c.context)

	newJWKS, err := jwk.NewJWKS(ctx, jsonData.JwksURI, c.OIDC.JWKsTTL, c.OIDC.JWKsMaxStale, jwksBackend)
	if err != nil { // do not replace possible working jwks on err
		cancel()
		return jsonData, err
	}

	// free up inner syncedJSON goroutine
	if c.jwksCancel != nil {
		c.jwksCancel()
	}
	// update with newly created one
	c.jwksCancel = cancel
	c.jwks = newJWKS

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
