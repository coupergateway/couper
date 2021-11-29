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

type JWKSData struct {
	Keys []JWK `json:"keys"`
}

type JWKS struct {
	syncedJSON *SyncedJSON
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
	sj := NewSyncedJSON(confContext, file, "jwks_url", uri, transport, "jwks" /* TODO which roundtrip name? */, timetolive, jwks)
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

type SyncedJSONUnmarshaller interface {
	Unmarshal(rawJSON []byte) (interface{}, error)
}

type SyncedJSON struct {
	context       context.Context
	file          string
	fileContext   string
	uri           string
	transport     http.RoundTripper
	roundTripName string
	ttl           time.Duration
	unmarshaller  SyncedJSONUnmarshaller
	// used internally
	data   interface{}
	expiry int64
	mtx    sync.RWMutex
}

func NewSyncedJSON(context context.Context, file, fileContext, uri string, transport http.RoundTripper, roundTripName string, ttl time.Duration, unmarshaller SyncedJSONUnmarshaller) *SyncedJSON {
	return &SyncedJSON{
		context:       context,
		file:          file,
		fileContext:   fileContext,
		uri:           uri,
		transport:     transport,
		roundTripName: roundTripName,
		ttl:           ttl,
		unmarshaller:  unmarshaller,
	}
}

func (s *SyncedJSON) Data() (interface{}, error) {
	var err error

	s.mtx.RLock()
	data := s.data
	expired := s.hasExpired()
	s.mtx.RUnlock()

	if data == nil || expired {
		data, err = s.Load()
		if err != nil {
			return nil, fmt.Errorf("error loading synced JSON: %v", err)
		}
	}

	return data, nil
}

func (s *SyncedJSON) Load() (interface{}, error) {
	var rawJSON []byte

	if s.file != "" {
		j, err := reader.ReadFromFile(s.fileContext, s.file)
		if err != nil {
			return nil, err
		}
		rawJSON = j
	} else if s.transport != nil {
		req, err := http.NewRequest("GET", "", nil)
		if err != nil {
			return nil, err
		}
		ctx := context.WithValue(s.context, request.URLAttribute, s.uri)
		ctx = context.WithValue(ctx, request.RoundTripName, s.roundTripName)
		req = req.WithContext(ctx)
		response, err := s.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("status code %d", response.StatusCode)
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response for %q: %v", s.uri, err)
		}
		rawJSON = body
	} else {
		return nil, fmt.Errorf("synced JSON: missing both file and request")
	}

	jsonData, err := s.unmarshaller.Unmarshal(rawJSON)
	if err != nil {
		return nil, err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.data = jsonData
	s.expiry = time.Now().Unix() + int64(s.ttl.Seconds())

	return jsonData, nil
}

func (s *SyncedJSON) hasExpired() bool {
	return time.Now().Unix() > s.expiry
}
