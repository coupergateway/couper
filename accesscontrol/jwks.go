package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jsn "github.com/avenga/couper/json"
)

type JWKSData struct {
	Keys []JWK `json:"keys"`
}

type JWKS struct {
	syncedJSON *jsn.SyncedJSON
}

func NewJWKS(uri string, ttl string, transport http.RoundTripper, confContext context.Context) (*JWKS, error) {
	if ttl == "" {
		ttl = "1h"
	}

	timetolive, err := time.ParseDuration(ttl)
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
	sj := jsn.NewSyncedJSON(confContext, file, "jwks_url", uri, transport, "jwks" /* TODO which roundtrip name? */, timetolive, jwks)
	jwks.syncedJSON = sj
	return jwks, nil
}

func (j *JWKS) GetKeys(kid string) ([]JWK, error) {
	var keys []JWK

	jwksData, err := j.Data()
	if err != nil {
		return nil, err
	}

	allKeys := jwksData.Keys
	for _, key := range allKeys {
		if key.KeyID == kid {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (j *JWKS) GetKey(kid string, alg string, use string) (*JWK, error) {
	keys, err := j.GetKeys(kid)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if key.Algorithm == alg && key.Use == use {
			return &key, nil
		}
	}
	return nil, nil
}

func (j *JWKS) Data() (*JWKSData, error) {
	data, err := j.syncedJSON.Data()
	if err != nil {
		return nil, err
	}

	jwksData, ok := data.(JWKSData)
	if !ok {
		return nil, fmt.Errorf("data not JWKS data: %#v", data)
	}

	return &jwksData, nil
}

func (j *JWKS) Unmarshal(rawJSON []byte) (interface{}, error) {
	var jsonData JWKSData
	err := json.Unmarshal(rawJSON, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
