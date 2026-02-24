package resource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/config/reader"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval/buffer"
)

// ResourceUnmarshaller defines the interface for unmarshalling fetched data.
// This interface works with any data format (JSON, XML, etc.).
type ResourceUnmarshaller interface {
	Unmarshal(raw []byte) (interface{}, error)
}

type dataRequest struct {
	obj interface{}
	err error
}

// SyncedResource provides automatic fetching and caching of remote data with TTL-based refresh.
// It works with any data format via the ResourceUnmarshaller interface.
// It's used for JWKS (JSON), SAML metadata (XML), OIDC configuration (JSON), and other synced data.
// The sync goroutine exits when the provided context is cancelled.
type SyncedResource struct {
	log           *logrus.Entry
	maxStale      time.Duration
	roundTripName string
	transport     http.RoundTripper
	ttl           time.Duration
	unmarshaller  ResourceUnmarshaller
	uri           string
	// used internally
	ctx         context.Context
	data        interface{}
	dataRequest chan chan *dataRequest
	fileMode    bool
}

func NewSyncedResource(
	ctx context.Context, file, fileContext, uri string, transport http.RoundTripper, roundTripName string,
	ttl time.Duration, maxStale time.Duration, unmarshaller ResourceUnmarshaller, log *logrus.Entry) (*SyncedResource, error) {
	sr := &SyncedResource{
		ctx:           ctx,
		dataRequest:   make(chan chan *dataRequest, 10),
		log:           log,
		maxStale:      maxStale,
		roundTripName: roundTripName,
		transport:     transport,
		ttl:           ttl,
		unmarshaller:  unmarshaller,
		uri:           uri,
	}

	if file != "" {
		if err := sr.readFile(fileContext, file); err != nil {
			return nil, err
		}
		sr.fileMode = true
	} else if transport != nil {
		// do not start go-routine on config check (-watch)
		if _, exist := ctx.Value(request.ConfigDryRun).(bool); !exist {
			go sr.sync()
		}
	} else {
		return nil, fmt.Errorf("synced resource: missing both file and request")
	}

	return sr, nil
}

func (s *SyncedResource) sync() {
	var expired <-chan time.Time
	var invalidated <-chan time.Time
	var backoff time.Duration

	init := func() {
		expired = time.After(s.ttl)
		backoff = time.Second
	}

	init()

	// Retry initial fetch with exponential backoff
	err := s.fetchWithRetry(3)
	if err != nil {
		if s.log != nil {
			s.log.WithField("url", s.uri).WithField("type", s.roundTripName).
				Warn("initial resource fetch failed, will keep retrying")
		}
		expired = time.After(0)
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-expired:
			err = s.fetch()
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
			if s.log != nil {
				s.log.WithField("url", s.uri).WithField("type", s.roundTripName).
					Warn("cached resource invalidated after max_stale expired")
			}
			s.data = nil
		}
	}
}

func (s *SyncedResource) Data() (interface{}, error) {
	if s.fileMode {
		return s.data, nil
	}

	rCh := make(chan *dataRequest, 1)

	select {
	case s.dataRequest <- rCh:
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}

	select {
	case result := <-rCh:
		return result.obj, result.err
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// fetchWithRetry attempts to fetch with exponential backoff retry
func (s *SyncedResource) fetchWithRetry(maxRetries int) error {
	var lastErr error
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		lastErr = s.fetch()
		if lastErr == nil {
			return nil
		}

		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		// Wait before retry (unless this is the last attempt)
		if i < maxRetries-1 {
			select {
			case <-time.After(backoff):
				backoff *= 2
				if backoff > time.Minute {
					backoff = time.Minute
				}
			case <-s.ctx.Done():
				return s.ctx.Err()
			}
		}
	}

	return lastErr
}

func (s *SyncedResource) fetch() error {
	req, _ := http.NewRequest("GET", s.uri, nil)

	outCtx, cancel := context.WithCancel(context.WithValue(s.ctx, request.RoundTripName, s.roundTripName))
	defer cancel()
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

func (s *SyncedResource) readFile(context, path string) error {
	raw, err := reader.ReadFromFile(context, path)
	if err != nil {
		return err
	}
	s.data, err = s.unmarshaller.Unmarshal(raw)
	return err
}
