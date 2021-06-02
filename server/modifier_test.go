package server_test

import (
	"net/http"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestIntegration_ResponseHeaders(t *testing.T) {
	const confFile = "testdata/integration/modifier/01_couper.hcl"

	client := newClient()

	type testCase struct {
		path       string
		expHeaders http.Header
	}

	for _, tc := range []testCase{
		{
			path: "/",
			expHeaders: http.Header{
				"X-Files":  []string{"true"},
				"X-Server": []string{"true"},
			},
		},
		{
			path: "/app",
			expHeaders: http.Header{
				"X-Server": []string{"true"},
				"X-Spa":    []string{"true"},
			},
		},
		{
			path: "/api",
			expHeaders: http.Header{
				"X-Api":      []string{"true"},
				"X-Endpoint": []string{"true"},
				"X-Server":   []string{"true"},
			},
		},
		{
			path:       "/fail",
			expHeaders: http.Header{},
		},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(t)

			shutdown, _ := newCouper(confFile, helper)
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			for n := range tc.expHeaders {
				if v := res.Header.Get(n); v != "true" {
					t.Errorf("Expected header not found: %s", n)
				}
			}
		})
	}
}
