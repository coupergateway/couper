package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	if uri[0:5] == "file:" {
		file = uri[5:]
	} else if uri[0:5] != "http:" && uri[0:6] != "https:" {
		return nil, fmt.Errorf("Unsupported JWKS URI scheme: %q", uri)
	}

	return &JWKS{
		context:   confContext,
		file:      file,
		uri:       uri,
		transport: transport,
		ttl:       timetolive,
	}, nil
}

func (self *JWKS) GetKeys(kid string) ([]JWK, error) {
	var keys []JWK

	if len(self.Keys) == 0 || self.hasExpired() {
		if err := self.Load(); err != nil {
			return keys, fmt.Errorf("Error loading JWKS: %v", err)
		}
	}

	for _, key := range self.Keys {
		if key.KeyID == kid {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (self *JWKS) GetKey(kid string, alg string, use string) (*JWK, error) {
	keys, err := self.GetKeys(kid)
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

func (self *JWKS) Load() error {
	var rawJSON []byte

	if self.file != "" {
		j, err := reader.ReadFromFile("jwks_url", self.file)
		if err != nil {
			return err
		}
		rawJSON = j
	} else if self.transport != nil {
		req, err := http.NewRequest("GET", "", nil)
		if err != nil {
			return err
		}
		ctx := context.WithValue(self.context, request.URLAttribute, self.uri)
		// TODO which roundtrip name?
		ctx = context.WithValue(ctx, request.RoundTripName, "jwks")
		req = req.WithContext(ctx)
		response, err := self.transport.RoundTrip(req)
		if err != nil {
			return err
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("Error reading JWKS response for %q: %v", self.uri, err)
		}
		rawJSON = body
	} else {
		return fmt.Errorf("JWKS: missing both file and request!")
	}

	var jwks JWKS
	err := json.Unmarshal(rawJSON, &jwks)
	if err != nil {
		return err
	}

	self.Keys = jwks.Keys
	self.expiry = time.Now().Unix() + int64(self.ttl.Seconds())

	return nil
}

func (jwks *JWKS) hasExpired() bool {
	return time.Now().Unix() > jwks.expiry
}
