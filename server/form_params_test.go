package server_test

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"testing"

	"github.com/coupergateway/couper/internal/test"
)

func TestIntegration_FormParams(t *testing.T) {
	client := newClient()

	const confPath = "testdata/integration/form_params/"

	type testCase struct {
		file    string
		method  string
		ct      string
		post    string
		expArgs string
		expCT   string
		expErr  string
	}

	for i, tc := range []testCase{
		{
			file:    "01_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y&d=d",
			expArgs: `"Args":{"a":["A"],"b":["B"],"c":["C 1","C 2"],"d":["d","D"],"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodDelete,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "expression evaluation error: form_params: method mismatch: DELETE",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPut,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"x":["X 1","X 2"],"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "expression evaluation error: form_params: method mismatch: PUT",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodGet,
			ct:      "text/plain",
			post:    "",
			expArgs: `"Args":{}`,
			expCT:   `"Content-Type":["text/plain"]`,
			expErr:  "expression evaluation error: form_params: method mismatch: GET",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodGet,
			ct:      "text/plain",
			post:    "not-supported",
			expArgs: `"Args":{}`,
			expCT:   `"Content-Type":["text/plain"]`,
			expErr:  `expression evaluation error: form_params: method mismatch: GET`,
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/foo",
			post:    "",
			expArgs: ``,
			expCT:   ``,
			expErr:  `expression evaluation error: form_params: content-type mismatch: application/foo`,
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodDelete,
			ct:      "application/x-www-form-urlencoded",
			post:    "",
			expArgs: ``,
			expCT:   ``,
			expErr:  `expression evaluation error: form_params: method mismatch: DELETE`,
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPut,
			ct:      "application/x-www-form-urlencoded",
			post:    "",
			expArgs: ``,
			expCT:   ``,
			expErr:  `expression evaluation error: form_params: method mismatch: PUT`,
		},
		{
			file:    "02_couper.hcl",
			method:  http.MethodGet,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "03_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "04_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
	} {
		t.Run("_"+tc.post, func(subT *testing.T) {
			helper := test.New(subT)

			shutdown, hook := newCouper(path.Join(confPath, tc.file), helper)
			defer shutdown()

			req, err := http.NewRequest(tc.method, "http://example.com:8080/", nil)
			helper.Must(err)

			req.Close = true

			if tc.post != "" {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte(tc.post)))
			}

			if tc.ct != "" {
				req.Header.Set("Content-Type", tc.ct)
			}

			hook.Reset()
			res, err := client.Do(req)
			helper.Must(err)

			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				subT.Errorf("%d: Expected status 200, given %d", i, res.StatusCode)
			}

			entries := hook.AllEntries()
			if len(entries) == 0 {
				subT.Fatal("Expected log messages, got none")
			}

			if entries[0].Message != tc.expErr {
				subT.Logf("%v", hook)
				subT.Errorf("%d: Expected message log: %s", i, tc.expErr)
			}

			resBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			if !bytes.Contains(resBytes, []byte(tc.expArgs)) {
				subT.Errorf("%d: \nwant: \n%s\nin: \n%s", i, tc.expArgs, resBytes)
			}

			if !bytes.Contains(resBytes, []byte(tc.expCT)) {
				subT.Errorf("%d: \nwant: \n%s\nin: \n%s", i, tc.expCT, resBytes)
			}
		})
	}
}
