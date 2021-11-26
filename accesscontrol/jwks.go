package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
)

type JWKS struct {
	Keys      []JWK `json:"keys"`
	context   context.Context
	expiry    int64
	file      string
	uri       string
	transport http.RoundTripper
	ttl       time.Duration
	mtx       sync.RWMutex
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

	return &JWKS{
		context:   confContext,
		file:      file,
		uri:       uri,
		transport: transport,
		ttl:       timetolive,
	}, nil
}

func (j *JWKS) GetKeys(kid string) ([]JWK, error) {
	var (
		keys []JWK
		err  error
	)

	j.mtx.RLock()
	allKeys := j.Keys
	expired := j.hasExpired()
	j.mtx.RUnlock()
	if len(allKeys) == 0 || expired {
		allKeys, err = j.Load()
		if err != nil {
			return keys, fmt.Errorf("error loading JWKS: %v", err)
		}
	}

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

func (j *JWKS) Load() ([]JWK, error) {
	var rawJSON []byte

	if j.file != "" {
		j, err := reader.ReadFromFile("jwks_url", j.file)
		if err != nil {
			return nil, err
		}
		rawJSON = j
	} else if j.transport != nil {
		req, err := http.NewRequest("GET", "", nil)
		if err != nil {
			return nil, err
		}
		ctx := context.WithValue(j.context, request.URLAttribute, j.uri)
		// TODO which roundtrip name?
		ctx = context.WithValue(ctx, request.RoundTripName, "jwks")
		req = req.WithContext(ctx)
		response, err := j.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("status code %d", response.StatusCode)
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading JWKS response for %q: %v", j.uri, err)
		}
		rawJSON = body
	} else {
		return nil, fmt.Errorf("jwks: missing both file and request")
	}

	var jwks JWKS
	err := json.Unmarshal(rawJSON, &jwks)
	if err != nil {
		return nil, err
	}

	j.mtx.Lock()
	defer j.mtx.Unlock()

	j.Keys = jwks.Keys
	j.expiry = time.Now().Unix() + int64(j.ttl.Seconds())

	return j.Keys, nil
}

func (jwks *JWKS) hasExpired() bool {
	return time.Now().Unix() > jwks.expiry
}
