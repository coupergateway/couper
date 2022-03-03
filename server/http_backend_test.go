package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

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
