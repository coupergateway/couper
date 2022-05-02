package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/internal/test"
)

func TestBackend_MaxConnections(t *testing.T) {
	helper := test.New(t)

	const reqCount = 3
	lastSeen := map[string]string{}
	lastSeenMu := sync.Mutex{}
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		lastSeenMu.Lock()
		defer lastSeenMu.Unlock()

		if lastSeen[r.URL.Path] != "" && lastSeen[r.URL.Path] != r.RemoteAddr {
			t.Errorf("expected same remote addr for path: %q", r.URL.Path)
			rw.WriteHeader(http.StatusInternalServerError)
		} else {
			rw.WriteHeader(http.StatusNoContent)
		}
		lastSeen[r.URL.Path] = r.RemoteAddr
	}))

	defer origin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/03_couper.hcl", helper, map[string]interface{}{
		"origin": origin.URL,
	})
	defer shutdown()

	paths := []string{
		"/",
		"/be",
		"/fake-sequence",
	}

	originWait := sync.WaitGroup{}
	originWait.Add(len(paths) * reqCount)
	waitForCh := make(chan struct{})

	client := test.NewHTTPClient()

	for _, clientPath := range paths {
		for i := 0; i < reqCount; i++ {
			go func(path string) {
				req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+path, nil)
				<-waitForCh
				res, err := client.Do(req)
				helper.Must(err)

				if res.StatusCode != http.StatusNoContent {
					t.Errorf("want: 204, got %d", res.StatusCode)
				}

				originWait.Done()
			}(clientPath)
		}
	}

	close(waitForCh)
	originWait.Wait()
}

func TestBackend_MaxConnections_BodyClose(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2) // always delay, ensures every req hit runs into max_conns issue

		rw.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(r.URL)
		helper.Must(err)
		_, err = rw.Write(b)
		helper.Must(err)
	}))

	defer origin.Close()

	shutdown, _ := newCouperWithTemplate("testdata/integration/backends/04_couper.hcl", helper,
		map[string]interface{}{
			"origin": origin.URL,
		})
	defer shutdown()

	client := test.NewHTTPClient()

	paths := []string{
		"/",
		"/named",
		"/default",
		"/default2",
		"/ws",
	}

	for _, p := range paths {
		deadline, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+p, nil)
		res, err := client.Do(req.WithContext(deadline))
		cancel()
		helper.Must(err)

		if res.StatusCode != http.StatusOK {
			t.Errorf("want: 200, got %d", res.StatusCode)
		}

		_, err = io.Copy(io.Discard, res.Body)
		helper.Must(err)

		helper.Must(res.Body.Close())
	}
}

// TestBackend_WithoutOrigin expects the listed errors to ensure no host from the client-request
// leaks into the backend structure for connecting to the origin.
func TestBackend_WithoutOrigin(t *testing.T) {
	helper := test.New(t)
	shutdown, hook := newCouper("testdata/integration/backends/01_couper.hcl", helper)
	defer shutdown()

	client := test.NewHTTPClient()

	for _, tc := range []struct {
		path    string
		message string
	}{
		{"/proxy/path", `configuration error: anonymous_6_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/proxy/backend-path", `configuration error: anonymous_15_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/proxy/url", `configuration error: anonymous_24_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/request/backend-path", `configuration error: anonymous_37_15: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
		{"/request/url", `configuration error: anonymous_46_13: the origin attribute has to contain an absolute URL with a valid hostname: ""`},
	} {
		t.Run(tc.path, func(st *testing.T) {
			hook.Reset()

			h := test.New(st)
			req, _ := http.NewRequest(http.MethodGet, "http://couper.dev:8080"+tc.path, nil)
			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != http.StatusInternalServerError {
				st.Errorf("want: 500, got %d", res.StatusCode)
			}

			for _, e := range hook.AllEntries() {
				if e.Level != logrus.ErrorLevel {
					continue
				}

				if e.Message != tc.message {
					st.Errorf("\nwant: %q\ngot:  %q\n", tc.message, e.Message)
				}
			}

		})

	}
}
