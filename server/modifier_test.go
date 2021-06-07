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

func TestIntegration_SetResponseStatus(t *testing.T) {
	const confFile = "testdata/integration/modifier/02_couper.hcl"

	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper(confFile, helper)
	defer shutdown()

	// ================================================== 204 >>>

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/204", nil)
	helper.Must(err)

	hook.Reset()
	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != 204 {
		t.Errorf("Expected status code 204, given: %d", res.StatusCode)
	}

	exp := "set_response_status sets the HTTP status code to 204 - removing the response body if any"
	if hook.Entries[0].Message != exp {
		t.Errorf("Unexpected message given: %s", hook.Entries[0].Message)
	}

	// ================================================== 201 >>>

	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/201", nil)
	helper.Must(err)

	hook.Reset()
	res, err = client.Do(req)
	helper.Must(err)

	if res.StatusCode != 201 {
		t.Errorf("Expected status code 201, given: %d", res.StatusCode)
	}
	if hook.Entries[0].Message != "" {
		t.Errorf("Unexpected message given: %s", hook.Entries[0].Message)
	}

	// ================================================== 600 >>>

	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/600", nil)
	helper.Must(err)

	hook.Reset()
	res, err = client.Do(req)
	helper.Must(err)

	if res.StatusCode != 500 {
		t.Errorf("Expected status code 500, given: %d", res.StatusCode)
	}

	exp = "configuration error: set_response_status sets an invalid HTTP status code: 600; set the status code to 500"
	if hook.Entries[0].Message != exp {
		t.Errorf("Unexpected message given: %s", hook.Entries[0].Message)
	}
}
