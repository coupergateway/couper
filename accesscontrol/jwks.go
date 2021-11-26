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
	var keys []JWK

	j.mtx.RLock()
	lKeys := len(j.Keys)
	j.mtx.RUnlock()
	if lKeys == 0 || j.hasExpired() {
		if err := j.Load(); err != nil {
			return keys, fmt.Errorf("error loading JWKS: %v", err)
		}
	}

	j.mtx.RLock()
	ks := j.Keys
	j.mtx.RUnlock()
	for _, key := range ks {
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

func (j *JWKS) Load() error {
	var rawJSON []byte

	if j.file != "" {
		j, err := reader.ReadFromFile("jwks_url", j.file)
		if err != nil {
			return err
		}
		rawJSON = j
	} else if j.transport != nil {
		req, err := http.NewRequest("GET", "", nil)
		if err != nil {
			return err
		}
		ctx := context.WithValue(j.context, request.URLAttribute, j.uri)
		// TODO which roundtrip name?
		ctx = context.WithValue(ctx, request.RoundTripName, "jwks")
		req = req.WithContext(ctx)
		response, err := j.transport.RoundTrip(req)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			return fmt.Errorf("status code %d", response.StatusCode)
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading JWKS response for %q: %v", j.uri, err)
		}
		rawJSON = body
	} else {
		return fmt.Errorf("jwks: missing both file and request")
	}

	var jwks JWKS
	err := json.Unmarshal(rawJSON, &jwks)
	if err != nil {
		return err
	}

	j.mtx.Lock()
	j.Keys = jwks.Keys
	j.expiry = time.Now().Unix() + int64(j.ttl.Seconds())
	j.mtx.Unlock()

	return nil
}

func (jwks *JWKS) hasExpired() bool {
	jwks.mtx.RLock()
	defer jwks.mtx.RUnlock()
	return time.Now().Unix() > jwks.expiry
}
