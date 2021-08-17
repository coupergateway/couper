package oidc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
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
	memStore   *cache.MemoryStore
	remoteConf *OpenidConfiguration
	remoteMu   sync.RWMutex
	ttl        int64
}

// NewConfig creates a new configuration for an OIDC client
func NewConfig(oidc *config.OIDC, backend http.RoundTripper, memStore *cache.MemoryStore) (*Config, error) {
	ttl := defaultTTL
	if oidc.TTL != "" {
		t, err := time.ParseDuration(oidc.TTL)
		if err != nil {
			return nil, err
		}
		ttl = t
	}

	return &Config{OIDC: oidc, Backend: backend, memStore: memStore, ttl: (int64)(ttl)}, nil
}

func (o *Config) Reference() string {
	return o.OIDC.BackendName
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or nonce)
func (o *Config) GetVerifierMethod(uid string) (string, error) {
	if o.VerifierMethod == "" {
		err := o.getFreshIfExpired(uid)
		if err != nil {
			return "", err
		}
	}

	return o.VerifierMethod, nil
}

func (o *Config) GetAuthorizationEndpoint(uid string) (string, error) {
	err := o.getFreshIfExpired(uid)
	if err != nil {
		return "", err
	}

	o.remoteMu.RLock()
	defer o.remoteMu.RUnlock()
	return o.remoteConf.AuthorizationEndpoint, nil
}

func (o *Config) GetIssuer() (string, error) {
	err := o.getFreshIfExpired("")
	if err != nil {
		return "", err
	}

	o.remoteMu.RLock()
	defer o.remoteMu.RUnlock()
	return o.remoteConf.Issuer, nil
}

func (o *Config) GetTokenEndpoint() (string, error) {
	err := o.getFreshIfExpired("")
	if err != nil {
		return "", err
	}

	o.remoteMu.RLock()
	defer o.remoteMu.RUnlock()
	return o.remoteConf.TokenEndpoint, nil
}

func (o *Config) GetUserinfoEndpoint() (string, error) {
	err := o.getFreshIfExpired("")
	if err != nil {
		return "", err
	}

	o.remoteMu.RLock()
	defer o.remoteMu.RUnlock()
	return o.remoteConf.UserinfoEndpoint, nil
}

func (o *Config) getFreshIfExpired(uid string) error {
	key := o.Name + o.ConfigurationURL
	confVal := o.memStore.Get(key)
	if oc, ok := confVal.(*OpenidConfiguration); ok {

		o.remoteMu.RLock()
		if oc.hash() == o.remoteConf.hash() {
			o.remoteMu.RUnlock()
			return nil
		}
		o.remoteMu.RUnlock()

		o.remoteMu.Lock()
		o.remoteConf = oc
		o.remoteMu.Unlock()
		return nil
	}

	conf, err := o.fetchConfiguration(uid)
	if err != nil {
		return err
	}

	o.remoteMu.Lock()
	defer o.remoteMu.Unlock()

	o.remoteConf = conf
	if o.OIDC.VerifierMethod == "" {
		if supportsS256(o.remoteConf.CodeChallengeMethodsSupported) {
			o.OIDC.VerifierMethod = config.CcmS256
		} else {
			o.OIDC.VerifierMethod = "nonce"
		}
	}

	return nil
}

func (o *Config) fetchConfiguration(uid string) (*OpenidConfiguration, error) {
	req, err := http.NewRequest(http.MethodGet, "", nil)
	ctx := context.WithValue(context.Background(), request.URLAttribute, o.ConfigurationURL)
	ctx = context.WithValue(ctx, request.RoundTripName, o.Name)
	if uid != "" {
		ctx = context.WithValue(ctx, request.UID, uid)
	}
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

	if len(ocBytes) == 0 {
		return nil, fmt.Errorf("configuration body is empty: %s", o.ConfigurationURL)
	}

	decoder := json.NewDecoder(bytes.NewReader(ocBytes))
	openidConfiguration := &OpenidConfiguration{}
	err = decoder.Decode(openidConfiguration)
	if err != nil {
		return nil, err
	}

	o.memStore.Set(o.ConfigurationURL, openidConfiguration, o.ttl)
	return openidConfiguration, nil
}

func (rc *OpenidConfiguration) hash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", rc)))
	return fmt.Sprintf("%x", h.Sum(nil))
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
