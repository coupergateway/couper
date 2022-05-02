package json

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
)

type SyncedJSONUnmarshaller interface {
	Unmarshal(rawJSON []byte) (interface{}, error)
}

type dataRequest struct {
	obj interface{}
	err error
}

type SyncedJSON struct {
	roundTripName string
	transport     http.RoundTripper
	ttl           time.Duration
	unmarshaller  SyncedJSONUnmarshaller
	uri           string
	// used internally
	data        interface{}
	dataRequest chan chan *dataRequest
	fileMode    bool
}

func NewSyncedJSON(file, fileContext, uri string, transport http.RoundTripper, roundTripName string, ttl time.Duration, unmarshaller SyncedJSONUnmarshaller) (*SyncedJSON, error) {
	sj := &SyncedJSON{
		dataRequest:   make(chan chan *dataRequest, 10),
		roundTripName: roundTripName,
		transport:     transport,
		ttl:           ttl,
		unmarshaller:  unmarshaller,
		uri:           uri,
	}

	if file != "" {
		if err := sj.readFile(fileContext, file); err != nil {
			return nil, err
		}
		sj.fileMode = true
	} else if transport != nil {
		go sj.sync(context.Background()) // TODO: at least cmd cancel ctx (reload)
	} else {
		return nil, fmt.Errorf("synced JSON: missing both file and request")
	}

	return sj, nil
}

func (s *SyncedJSON) sync(ctx context.Context) {
	var expired <-chan time.Time
	err := s.fetch() // initial fetch, provide any startup errors for first dataRequest's
	if err != nil {
		expired = time.After(0)
	} else {
		expired = time.After(s.ttl)
	}

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-expired:
			err = s.fetch()
			if err != nil {
				time.Sleep(backoff)
				if backoff < time.Minute {
					backoff *= 2
				}
				continue
			}
			expired = time.After(s.ttl)
			backoff = time.Second
		case r := <-s.dataRequest:
			r <- &dataRequest{
				err: err,
				obj: s.data,
			}
		}
	}
}

func (s *SyncedJSON) Data() (interface{}, error) {
	if s.fileMode {
		return s.data, nil
	}

	rCh := make(chan *dataRequest)
	s.dataRequest <- rCh
	result := <-rCh
	return result.obj, result.err
}

// fetch blocks all data reads until we will have an updated one.
func (s *SyncedJSON) fetch() error {
	req, _ := http.NewRequest("GET", s.uri, nil)

	ctx := context.WithValue(context.Background(), request.RoundTripName, s.roundTripName)
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
