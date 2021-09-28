package accesscontrol

import (
	"encoding/json"
	"fmt"
	"github.com/avenga/couper/config/reader"
	"time"
)

type JWKS struct {
	Keys []JWK `json:"keys"`
	uri  string
	ttl  time.Duration
}

func NewJWKS(uri string, ttl string) (*JWKS, error) {
	if ttl == "" {
		ttl = "1h"
	}

	timetolive, err := time.ParseDuration(ttl)

	if err != nil {
		return nil, err
	}

	return &JWKS{
		uri: uri,
		ttl: timetolive,
	}, nil
}

func (self *JWKS) GetKeys(kid string) []JWK {
	var keys []JWK

	if err := self.Load(); err != nil {
		fmt.Printf("Error loading JWKS: %v\n", err)
		return keys
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
	// TODO Lookup cache

	var rawJSON []byte
	if self.uri[0:5] == "file:" {
		filename := self.uri[5:]
		j, err := reader.ReadFromFile("jwks_uri", filename)
		if err != nil {
			return err
		}
		rawJSON = j
	} else if self.uri[0:5] == "http:" || self.uri[0:6] == "https:" {
		// TODO HTTP request
		return fmt.Errorf("Could not fetch JWKS from %q", self.uri)
	} else {
		return fmt.Errorf("Unsupported JWKS URI scheme: %q", self.uri)
	}

	var jwks JWKS
	err := json.Unmarshal(rawJSON, &jwks)
	if err != nil {
		return err
	}

	self.Keys = jwks.Keys

	// TODO store in cache
	return nil
}
