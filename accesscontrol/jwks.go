package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avenga/couper/config/reader"
	"io/ioutil"
	"net/http"
	"time"
)

type JWKS struct {
	Keys      []JWK `json:"keys"`
	context   context.Context
	expiry    int64
	file      string
	request   *http.Request
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
	var request *http.Request
	if uri[0:5] == "file:" {
		file = uri[5:]
	} else if uri[0:5] == "http:" || uri[0:6] == "https:" {
		r, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return nil, err
		}
		request = r
	} else {
		return nil, fmt.Errorf("Unsupported JWKS URI scheme: %q", uri)
	}

	return &JWKS{
		context:   confContext,
		file:      file,
		request:   request,
		uri:       uri,
		transport: transport,
		ttl:       timetolive,
	}, nil
}

func (self *JWKS) GetKeys(kid string) []JWK {
	var keys []JWK

	if len(self.Keys) == 0 || self.hasExpired() {
		if err := self.Load(); err != nil {
			fmt.Printf("Error loading JWKS: %v\n", err)
			return keys
		}
	}

	for _, key := range self.Keys {
		if key.KeyID == kid || kid == "" {
			keys = append(keys, key)
		}
	}

	return keys
}

func (self *JWKS) GetKey(kid string, alg string, use string) *JWK {
	for _, key := range self.GetKeys(kid) {
		if key.Algorithm == alg && key.Use == use {
			return &key
		}
	}
	return nil
}

func (self *JWKS) Load() error {
	var rawJSON []byte

	if self.file != "" {
		j, err := reader.ReadFromFile("jwks_uri", self.file)
		if err != nil {
			return err
		}
		rawJSON = j
	} else if self.request != nil && self.transport != nil {
		request := self.request.WithContext(self.context)
		response, err := self.transport.RoundTrip(request)
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
