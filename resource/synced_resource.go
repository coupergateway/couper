package resource

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
type SyncedResource struct {
	maxStale      time.Duration
	roundTripName string
	transport     http.RoundTripper
	ttl           time.Duration
	unmarshaller  ResourceUnmarshaller
	uri           string
	// used internally
	data        interface{}
	dataRequest chan chan *dataRequest
	fileMode    bool
	stopCh      chan struct{} // Explicit stop signal
	stoppedCh   chan struct{} // Confirmation of stop
}

func NewSyncedResource(
	ctx context.Context, file, fileContext, uri string, transport http.RoundTripper, roundTripName string,
	ttl time.Duration, maxStale time.Duration, unmarshaller ResourceUnmarshaller) (*SyncedResource, error) {
	sr := &SyncedResource{
		dataRequest:   make(chan chan *dataRequest, 10),
		maxStale:      maxStale,
		roundTripName: roundTripName,
		transport:     transport,
		ttl:           ttl,
		unmarshaller:  unmarshaller,
		uri:           uri,
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	if file != "" {
		if err := sr.readFile(fileContext, file); err != nil {
			return nil, err
		}
		sr.fileMode = true
	} else if transport != nil {
		// do not start go-routine on config check (-watch)
		if _, exist := ctx.Value(request.ConfigDryRun).(bool); !exist {
			go sr.sync(ctx)
		}
	} else {
		return nil, fmt.Errorf("synced resource: missing both file and request")
	}

	return sr, nil
}

func (s *SyncedResource) sync(ctx context.Context) {
	defer close(s.stoppedCh)

	var expired <-chan time.Time
	var invalidated <-chan time.Time
	var backoff time.Duration

	init := func() {
		expired = time.After(s.ttl)
		backoff = time.Second
	}

	init()

	// Retry initial fetch with exponential backoff
	err := s.fetchWithRetry(ctx, 3) // Try up to 3 times on initial fetch
	if err != nil {
		expired = time.After(0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
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

// Stop gracefully stops the sync goroutine
func (s *SyncedResource) Stop() {
	if s.fileMode {
		return
	}
	close(s.stopCh)
	<-s.stoppedCh
}

func (s *SyncedResource) Data() (interface{}, error) {
	if s.fileMode {
		return s.data, nil
	}

	rCh := make(chan *dataRequest)

	// Use a timeout to prevent indefinite blocking if sync goroutine is dead
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case s.dataRequest <- rCh:
		select {
		case result := <-rCh:
			return result.obj, result.err
		case <-timer.C:
			return nil, fmt.Errorf("timeout waiting for data from sync goroutine")
		}
	case <-timer.C:
		return nil, fmt.Errorf("timeout sending data request to sync goroutine")
	}
}

// fetchWithRetry attempts to fetch with exponential backoff retry
func (s *SyncedResource) fetchWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		lastErr = s.fetch(ctx)
		if lastErr == nil {
			return nil
		}

		// Don't retry if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return fmt.Errorf("sync stopped")
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
			case <-ctx.Done():
				return ctx.Err()
			case <-s.stopCh:
				return fmt.Errorf("sync stopped")
			}
		}
	}

	return lastErr
}

// fetch blocks all data reads until we will have an updated one.
func (s *SyncedResource) fetch(ctx context.Context) error {
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

func (s *SyncedResource) readFile(context, path string) error {
	raw, err := reader.ReadFromFile(context, path)
	if err != nil {
		return err
	}
	s.data, err = s.unmarshaller.Unmarshal(raw)
	return err
}
