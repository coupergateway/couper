package jwk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/resource"
)

var alg2kty = map[string]string{
	"RS256": "RSA",
	"RS384": "RSA",
	"RS512": "RSA",
	"ES256": "EC",
	"ES384": "EC",
	"ES512": "EC",
}

type JWKSData struct {
	Keys []*JWK `json:"keys"`
}

type JWKS struct {
	syncedResource *resource.SyncedResource
}

func NewJWKS(ctx context.Context, uri string, ttl string, maxStale string, transport http.RoundTripper, log *logrus.Entry) (*JWKS, error) {
	timetolive, err := config.ParseDuration("jwks_ttl", ttl, time.Hour)
	if err != nil {
		return nil, err
	}
	maxStaleTime, err := config.ParseDuration("jwks_max_stale", maxStale, time.Hour)
	if err != nil {
		return nil, err
	}
	var file string
	if strings.HasPrefix(uri, "file:") {
		file = uri[5:]
	} else if !strings.HasPrefix(uri, "http:") && !strings.HasPrefix(uri, "https:") {
		return nil, fmt.Errorf("unsupported JWKS URI scheme: %q", uri)
	}

	jwks := &JWKS{}
	jwks.syncedResource, err = resource.NewSyncedResource(ctx, file, "jwks_url", uri, transport, "jwks", timetolive, maxStaleTime, jwks, log)
	return jwks, err
}

func (j *JWKS) GetSigKeyForToken(token *jwt.Token) (interface{}, error) {
	algorithm := token.Header["alg"]
	if algorithm == nil {
		return nil, fmt.Errorf("missing \"alg\" in JOSE header")
	}
	id := token.Header["kid"]
	if id == nil {
		id = ""
	}
	jwk, err := j.GetKey(id.(string), algorithm.(string), "sig")
	if err != nil {
		return nil, err
	}

	if jwk == nil {
		return nil, fmt.Errorf("no matching %s JWK for kid %q", algorithm, id)
	}

	return jwk.Key, nil
}

func (j *JWKS) GetKey(kid string, alg string, use string) (*JWK, error) {
	keys, err := j.getKeys(kid)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Use == use {
			if key.Algorithm == alg {
				return key, nil
			} else if key.Algorithm == "" {
				if kty, exists := alg2kty[alg]; exists && key.KeyType == kty {
					return key, nil
				}
			}
		}
	}
	return nil, nil
}

func (j *JWKS) getKeys(kid string) ([]*JWK, error) {
	var keys []*JWK

	jwksData, err := j.Data()
	if err != nil {
		return nil, err
	}

	if len(jwksData.Keys) == 0 {
		return nil, fmt.Errorf("missing jwks key-data")
	}

	for _, key := range jwksData.Keys {
		if key.KeyID == kid {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (j *JWKS) Data() (*JWKSData, error) {
	data, err := j.syncedResource.Data()
	// Ignore backend errors as long as we still get cached (stale) data.
	jwksData, ok := data.(*JWKSData)
	if !ok {
		return nil, fmt.Errorf("received no valid JWKs data: %#v, %w", data, err)
	}

	return jwksData, nil
}

func (j *JWKS) Unmarshal(raw []byte) (interface{}, error) {
	jsonData := &JWKSData{}
	err := json.Unmarshal(raw, jsonData)
	return jsonData, err
}
