package resource_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/resource"
)

type unmarshaller struct {
	resource.SyncedResource
}

type data struct {
	Foo int `json:"foo"`
}

func (u *unmarshaller) Unmarshal(raw []byte) (interface{}, error) {
	jsonData := &data{}
	err := json.Unmarshal(raw, jsonData)
	return jsonData, err
}

func Test_LoadSynced(t *testing.T) {
	helper := test.New(t)

	requestCount := 0
	origin := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		if requestCount == 2 || requestCount == 3 || requestCount == 4 {
			response.WriteHeader(http.StatusInternalServerError)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		response.Write([]byte(`{"foo": ` + fmt.Sprintf("%d", requestCount) + `}`))
	}))
	defer origin.Close()

	syncedResource, err := resource.NewSyncedResource(context.TODO(), "", "", origin.URL, http.DefaultTransport, "test", time.Second*2, time.Hour, &unmarshaller{}, nil)
	helper.Must(err)

	expectJSONValue := func(expectedValue int, shouldFail bool) {
		o, err := syncedResource.Data()
		if err == nil && shouldFail {
			t.Fatalf("expected sync to fail - backoff too small!?")
		} else if err != nil && !shouldFail {
			t.Fatalf("unexpected sync failure - backoff too large? %v", err)
		}

		if o == nil {
			t.Fatalf("expected JSON response, got: nil")
		}

		object := o.(*data)
		if object.Foo != expectedValue {
			t.Fatalf("unexpected JSON value, want: %d, got: %v", expectedValue, object.Foo)
		}
	}

	// First request, JSON response cached for 2s
	for i := 0; i < 2; i++ {
		expectJSONValue(1, false)
		time.Sleep(time.Millisecond * 900)
	}

	time.Sleep(time.Millisecond * 300)

	// TTL (2s) has expired --> 2nd request fails!
	// Subsequent requests after 1s (3rd request) and after 3s (4th) fail, too:
	// --> old JSON is returned for 7s
	for i := 0; i < 7; i++ {
		expectJSONValue(1, true)
		time.Sleep(time.Second)
	}

	// another 4s later (7s after expiry), the 5th request finally succeeds:
	expectJSONValue(5, false)
	time.Sleep(2100 * time.Millisecond)
	// After 2 more seconds, refresh again:
	expectJSONValue(6, false)
}

func Test_SyncedResource_ContextCancellation(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"foo": 1}`))
	}))
	defer origin.Close()

	ctx, cancel := context.WithCancel(context.Background())

	sr, err := resource.NewSyncedResource(ctx, "", "", origin.URL, http.DefaultTransport, "test", time.Hour, time.Hour, &unmarshaller{}, nil)
	helper.Must(err)

	// Should work before cancellation
	obj, err := sr.Data()
	helper.Must(err)
	if obj.(*data).Foo != 1 {
		t.Fatalf("expected foo=1, got %v", obj)
	}

	// Cancel context
	cancel()

	// Give sync goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// Data() should return context error
	_, err = sr.Data()
	if err == nil {
		t.Fatal("expected error after context cancellation")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func Test_SyncedResource_InitialFetchRetry(t *testing.T) {
	helper := test.New(t)

	requestCount := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// First 2 requests fail, 3rd succeeds
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"foo": 42}`))
	}))
	defer origin.Close()

	sr, err := resource.NewSyncedResource(context.Background(), "", "", origin.URL, http.DefaultTransport, "test", time.Hour, time.Hour, &unmarshaller{}, nil)
	helper.Must(err)

	// Despite initial failures, retry should succeed
	obj, err := sr.Data()
	helper.Must(err)

	if obj.(*data).Foo != 42 {
		t.Fatalf("expected foo=42, got %v", obj)
	}

	// Should have retried (3 requests total for initial fetch)
	if requestCount < 3 {
		t.Fatalf("expected at least 3 requests due to retry, got %d", requestCount)
	}
}

func Test_SyncedResource_MaxStaleInvalidation(t *testing.T) {
	helper := test.New(t)

	requestCount := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// First request succeeds, all subsequent fail
		if requestCount > 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"foo": 1}`))
	}))
	defer origin.Close()

	log, hook := logrustest.NewNullLogger()
	logEntry := log.WithField("test", true)

	// Short TTL (200ms) and short max_stale (500ms)
	sr, err := resource.NewSyncedResource(context.Background(), "", "", origin.URL, http.DefaultTransport, "test_type",
		200*time.Millisecond, 500*time.Millisecond, &unmarshaller{}, logEntry)
	helper.Must(err)

	// Initial fetch should succeed
	obj, err := sr.Data()
	helper.Must(err)
	if obj.(*data).Foo != 1 {
		t.Fatalf("expected foo=1, got %v", obj)
	}

	// Wait for TTL to expire + fetch failures + max_stale to expire
	// TTL=200ms, then retries fail, max_stale=500ms from first failure
	time.Sleep(1500 * time.Millisecond)

	// After max_stale expires, data should be nil
	obj, err = sr.Data()
	if obj != nil {
		t.Fatalf("expected nil data after max_stale expiry, got: %v", obj)
	}

	// Check that the invalidation warning was logged
	found := false
	for _, entry := range hook.AllEntries() {
		if entry.Message == "cached resource invalidated after max_stale expired" {
			found = true
			if entry.Data["type"] != "test_type" {
				t.Errorf("expected type=test_type, got %v", entry.Data["type"])
			}
			break
		}
	}
	if !found {
		t.Error("expected 'cached resource invalidated after max_stale expired' warning log")
	}
}
