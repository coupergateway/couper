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
	roundTripName string
	transport     http.RoundTripper
	ttl           time.Duration
	unmarshaller  SyncedJSONUnmarshaller
	uri           string
	// used internally
	data    interface{}
	dataErr chan error
	dataMu  sync.RWMutex
}

func NewSyncedJSON(file, fileContext, uri string, transport http.RoundTripper, roundTripName string, ttl time.Duration, unmarshaller SyncedJSONUnmarshaller) (*SyncedJSON, error) {
	sj := &SyncedJSON{
		dataErr:       make(chan error, 1),
		roundTripName: roundTripName,
		transport:     transport,
		ttl:           ttl,
		unmarshaller:  unmarshaller,
		uri:           uri,
	}

	var err error
	if file != "" {
		if err := sj.readFile(fileContext, file); err != nil {
			return nil, err
		}
	} else if transport != nil {
		err = sj.fetch() // initial fetch
		if err == nil {
			go sj.sync(context.Background()) // TODO: at least cmd cancel (reload)
		}
	} else {
		return nil, fmt.Errorf("synced JSON: missing both file and request")
	}

	return sj, err
}

func (s *SyncedJSON) sync(ctx context.Context) {
	expired := time.After(0) // force initial fetch()
	for {
		select {
		case <-ctx.Done():
			return
		case <-expired:
			err := s.fetch()
			if err != nil {
				select {
				case s.dataErr <- err:
				default:
				}
				time.Sleep(time.Second * 1)
				continue
			}
			expired = time.After(s.ttl)
		}
	}
}

func (s *SyncedJSON) Data(uid string) (interface{}, error) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	var err error
	select {
	case err = <-s.dataErr:
	default:
	}
	return s.data, err
}

// fetch blocks all data reads until we will have an updated one.
func (s *SyncedJSON) fetch() error {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	req, _ := http.NewRequest("GET", s.uri, nil)

	ctx := context.WithValue(context.Background(), request.URLAttribute, s.uri)
	ctx = context.WithValue(ctx, request.RoundTripName, s.roundTripName)

	req = req.WithContext(ctx)

	response, err := s.transport.RoundTrip(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", response.StatusCode)
	}

	defer response.Body.Close()

	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response for %q: %v", s.uri, err)
	}

	s.data, err = s.unmarshaller.Unmarshal(raw)
	return err
}

func (s *SyncedJSON) readFile(context, path string) error {
	raw, err := reader.ReadFromFile(context, path)
	if err != nil {
		return err
	}
	s.data, err = s.unmarshaller.Unmarshal(raw)
	return err
}
