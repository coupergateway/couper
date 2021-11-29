package json

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
)

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
