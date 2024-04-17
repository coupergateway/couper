package json_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
	jsn "github.com/coupergateway/couper/json"
)

type unmarshaller struct {
	jsn.SyncedJSON
}

type data struct {
	Foo int `json:"foo"`
}

func (u *unmarshaller) Unmarshal(rawJSON []byte) (interface{}, error) {
	jsonData := &data{}
	err := json.Unmarshal(rawJSON, jsonData)
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

	syncedJSON, err := jsn.NewSyncedJSON(context.TODO(), "", "", origin.URL, http.DefaultTransport, "test", time.Second*2, time.Hour, &unmarshaller{})
	helper.Must(err)

	expectJSONValue := func(expectedValue int, shouldFail bool) {
		o, err := syncedJSON.Data()
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
