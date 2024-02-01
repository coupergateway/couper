package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
)

func TestIntegration_ResponseHeaders(t *testing.T) {
	const confFile = "testdata/integration/modifier/01_couper.hcl"

	shutdown, _ := newCouper(confFile, test.New(t))
	defer shutdown()

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
			path:       "/not-found",
			expHeaders: http.Header{},
		},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			for n := range tc.expHeaders {
				if v := res.Header.Get(n); v != "true" {
					subT.Errorf("Expected header not found: %s", n)
				}
			}
		})
	}
}

func TestIntegration_SetResponseStatus(t *testing.T) {
	const confFile = "testdata/integration/modifier/02_couper.hcl"

	shutdown, hook := newCouper(confFile, test.New(t))
	defer shutdown()

	client := newClient()

	type testCase struct {
		path       string
		expMessage string
		expStatus  int
	}

	for _, tc := range []testCase{
		{
			path:       "/204",
			expMessage: "set_response_status: removing body, if any due to status-code 204",
			expStatus:  http.StatusNoContent,
		},
		{
			path:       "/201",
			expMessage: "",
			expStatus:  http.StatusCreated,
		},
		{
			path:       "/600",
			expMessage: "configuration error: set_response_status: invalid http status code: 600",
			expStatus:  http.StatusInternalServerError,
		},
		{
			path:       "/teapot",
			expMessage: "access control error: ba: credentials required",
			expStatus:  http.StatusTeapot,
		},
		{
			path:       "/no-content",
			expMessage: "", // logs without err have no/an empty message field
			expStatus:  http.StatusNoContent,
		},
		{
			path:       "/happy-path-only",
			expMessage: `backend error: anonymous_40_11: unsupported protocol scheme "couper"`,
			expStatus:  http.StatusBadGateway,
		},
		{
			path:       "/inception",
			expMessage: `backend error: anonymous_64_15: unsupported protocol scheme "couper"`,
			expStatus:  http.StatusBadGateway,
		},
	} {
		t.Run(tc.path, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://example.com:8080"+tc.path, nil)
			helper.Must(err)

			hook.Reset()
			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				subT.Errorf("Expected status code %d, given: %d", tc.expStatus, res.StatusCode)
			}

			time.Sleep(time.Second / 2) // MAYbe entries arent written yet
			for _, entry := range hook.AllEntries() {
				if entry.Message == tc.expMessage {
					return
				}
			}

			subT.Errorf("expected log message not seen: %s\ngot: %s", tc.expMessage, hook.LastEntry().Message)
		})
	}
}
