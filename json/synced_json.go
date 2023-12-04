package json

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coupergateway/couper/config/reader"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval/buffer"
)

type SyncedJSONUnmarshaller interface {
	Unmarshal(rawJSON []byte) (interface{}, error)
}

type dataRequest struct {
	obj interface{}
	err error
}

type SyncedJSON struct {
	maxStale      time.Duration
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

func NewSyncedJSON(
	ctx context.Context, file, fileContext, uri string, transport http.RoundTripper, roundTripName string,
	ttl time.Duration, maxStale time.Duration, unmarshaller SyncedJSONUnmarshaller) (*SyncedJSON, error) {
	sj := &SyncedJSON{
		dataRequest:   make(chan chan *dataRequest, 10),
		maxStale:      maxStale,
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
		// do not start go-routine on config check (-watch)
		if _, exist := ctx.Value(request.ConfigDryRun).(bool); !exist {
			go sj.sync(ctx)
		}
	} else {
		return nil, fmt.Errorf("synced JSON: missing both file and request")
	}

	return sj, nil
}

func (s *SyncedJSON) sync(ctx context.Context) {
	var expired <-chan time.Time
	var invalidated <-chan time.Time
	var backoff time.Duration

	init := func() {
		expired = time.After(s.ttl)
		backoff = time.Second
	}

	init()

	err := s.fetch(ctx) // initial fetch, provide any startup errors for first dataRequests
	if err != nil {
		expired = time.After(0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-expired:
			err = s.fetch(ctx)
			if err != nil {
				invalidated = time.After(s.maxStale)
				expired = time.After(backoff)
				if backoff < time.Minute {
					backoff *= 2
				}
				continue
			}
			init()
		case r := <-s.dataRequest:
			r <- &dataRequest{
				err: err,
				obj: s.data,
			}
		case <-invalidated:
			s.data = nil
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
func (s *SyncedJSON) fetch(ctx context.Context) error {
	req, _ := http.NewRequest("GET", s.uri, nil)

	outCtx, cancel := context.WithCancel(context.WithValue(ctx, request.RoundTripName, s.roundTripName))
	defer cancel()
	// Set the buffer option otherwise the resp body gets closed due to a non default roundtrip name. Have to read it anyway.
	outCtx = context.WithValue(outCtx, request.BufferOptions, buffer.Option(buffer.Response|buffer.JSONParseResponse))

	req = req.WithContext(outCtx)

	response, err := s.transport.RoundTrip(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", response.StatusCode)
	}

	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
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
