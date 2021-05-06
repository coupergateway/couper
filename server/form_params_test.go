package server_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/avenga/couper/internal/test"
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
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"a":["A"],"b":["B"],"c":["C 1","C 2"],"d":["D"],"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPut,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"a":["A"],"b":["B"],"c":["C 1","C 2"],"d":["D"],"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPatch,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"a":["A"],"b":["B"],"c":["C 1","C 2"],"d":["D"],"y":["Y"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodGet,
			ct:      "text/plain",
			post:    "",
			expArgs: `"Args":{"a":["A"],"b":["B"],"c":["C 1","C 2"],"d":["D"]}`,
			expCT:   `"Content-Type":["application/x-www-form-urlencoded"]`,
			expErr:  "",
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodGet,
			ct:      "text/plain",
			post:    "not-supported",
			expArgs: ``,
			expCT:   ``,
			expErr:  `request error: form_params: cannot apply form_params to a non-empty body within a GET request`,
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/foo",
			post:    "",
			expArgs: ``,
			expCT:   ``,
			expErr:  `request error: form_params: content type mismatch`,
		},
		{
			file:    "01_couper.hcl",
			method:  http.MethodDelete,
			ct:      "application/x-www-form-urlencoded",
			post:    "",
			expArgs: ``,
			expCT:   ``,
			expErr:  `request error: form_params: method missmatch: DELETE`,
		},
		{
			file:    "02_couper.hcl",
			method:  http.MethodGet,
			ct:      "text/plain",
			post:    "",
			expArgs: `"Args":{}`,
			expCT:   `"Content-Type":["text/plain"]`,
			expErr:  "",
		},
		{
			file:    "03_couper.hcl",
			method:  http.MethodPost,
			ct:      "application/x-www-form-urlencoded",
			post:    "x=X+1&x=X%202&y=Y",
			expArgs: `"Args":{"x":["X 1","X 2"],"y":["Y"]}`,
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

			req.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(tc.post)))
			if tc.ct != "" {
				req.Header.Set("Content-Type", tc.ct)
			}

			hook.Reset()
			res, err := client.Do(req)
			helper.Must(err)

			if tc.expErr != "" {
				if res.StatusCode != http.StatusBadRequest {
					t.Fatalf("%d: Expected status 400, given %d", i, res.StatusCode)
				}

				if hook.LastEntry().Message != tc.expErr {
					t.Logf("%v", hook.LastEntry())
					t.Errorf("%d: Expected message log: %s", i, tc.expErr)
				}
			} else {
				if res.StatusCode != http.StatusOK {
					t.Fatalf("%d: Expected status 200, given %d", i, res.StatusCode)
				}

				resBytes, err := ioutil.ReadAll(res.Body)
				helper.Must(err)

				_ = res.Body.Close()

				if !bytes.Contains(resBytes, []byte(tc.expArgs)) {
					t.Errorf("%d: \nwant: \n%s\nin: \n%s", i, tc.expArgs, resBytes)
				}

				if !bytes.Contains(resBytes, []byte(tc.expCT)) {
					t.Errorf("%d: \nwant: \n%s\nin: \n%s", i, tc.expCT, resBytes)
				}
			}
		})
	}
}
