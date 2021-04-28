package server_test

import (
	"net/http"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestAccessControl_ErrorHandler(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/error_handler/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name          string
		header        test.Header
		expLogMsg     string
		expStatusCode int
	}

	for _, tc := range []testCase{
		{"catch all", test.Header{"Authorization": "Basic aGFuczpoYW5z"}, "access control error: ba", http.StatusNotFound},
		{"catch specific", nil, "access control error: ba: missing credentials", http.StatusBadGateway},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			helper.Must(err)

			tc.header.Set(req)

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != tc.expStatusCode {
				t.Errorf("%q: expected Status %d, got: %d", tc.name, tc.expStatusCode, res.StatusCode)
				return
			}

			if logHook.LastEntry().Data["status"] != tc.expStatusCode {
				t.Logf("%v", logHook.LastEntry())
				t.Errorf("Expected statusCode log: %d", tc.expStatusCode)
			}

			if logHook.LastEntry().Message != tc.expLogMsg {
				t.Logf("%v", logHook.LastEntry())
				t.Errorf("Expected message log: %s", tc.expLogMsg)
			}
		})
	}
}

func TestAccessControl_ErrorHandler_BasicAuth_Default(t *testing.T) {
	client := newClient()

	shutdown, _ := newCouper("testdata/integration/error_handler/01_couper.hcl", test.New(t))
	defer shutdown()

	helper := test.New(t)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/default/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected Status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
		return
	}

	if www := res.Header.Get("www-authenticate"); www != "Basic realm=protected" {
		t.Errorf("Expected header: www-authenticate with value: %s, got: %s", "Basic realm=protected", www)
	}
}
