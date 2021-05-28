package server_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/internal/test"
)

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)

	for i := range []int{0, 1, 2} {
		var seenCh, tokenSeenCh chan struct{}

		retries := 0

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/oauth2" {
				if cl := req.Header.Get("Content-Length"); cl != "29" {
					t.Errorf("Unexpected C/L given: %s", cl)
				}

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)

				body := []byte(`{
					"access_token": "abcdef0123456789",
					"token_type": "bearer",
					"expires_in": 100
				}`)
				_, werr := rw.Write(body)
				helper.Must(werr)

				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if retries == i {
					close(tokenSeenCh)
				}

				return
			}
			rw.WriteHeader(http.StatusBadRequest)
		}))
		defer oauthOrigin.Close()

		ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/resource" {
				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if req.Header.Get("Authorization") == "Bearer abcdef0123456789" && retries == i {
					rw.WriteHeader(http.StatusNoContent)
					close(seenCh)
					return
				}

				retries++

				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			rw.WriteHeader(http.StatusNotFound)
		}))
		defer ResourceOrigin.Close()

		confPath := fmt.Sprintf("testdata/oauth2/%d_retries_couper.hcl", i)
		shutdown, hook := newCouper(confPath, test.New(t))
		defer func() {
			if t.Failed() {
				for _, e := range hook.Entries {
					println(e.String())
				}
			}
			shutdown()
		}()

		req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
		helper.Must(err)

		req.Header.Set("X-Token-Endpoint", oauthOrigin.URL)
		req.Header.Set("X-Origin", ResourceOrigin.URL)

		for _, p := range []string{"/", "/2nd"} {
			hook.Reset()

			seenCh = make(chan struct{})
			tokenSeenCh = make(chan struct{})

			req.URL.Path = p
			res, err := newClient().Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("expected status NoContent, got: %d", res.StatusCode)
				return
			}

			timer := time.NewTimer(time.Second * 2)
			select {
			case <-timer.C:
				t.Error("OAuth2 request failed")
			case <-tokenSeenCh:
				<-seenCh
			}
		}

		oauthOrigin.Close()
		ResourceOrigin.Close()
		shutdown()
	}
}

func TestEndpoints_OAuth2_Options(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		configFile string
		expBody    string
		expAuth    string
	}

	for _, tc := range []testCase{
		{
			"01_couper.hcl",
			`client_id=user&client_secret=pass+word&grant_type=client_credentials&scope=scope1+scope2`,
			"",
		},
		{
			"02_couper.hcl",
			`grant_type=client_credentials`,
			"Basic dXNlcjpwYXNz",
		},
		{
			"03_couper.hcl",
			`grant_type=client_credentials`,
			"Basic dXNlcjpwYXNz",
		},
	} {
		var tokenSeenCh chan struct{}

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/options" {
				reqBody, _ := ioutil.ReadAll(req.Body)
				authorization := req.Header.Get("Authorization")

				if tc.expBody != string(reqBody) {
					t.Errorf("want\n%s\ngot\n%s", tc.expBody, reqBody)
				}
				if tc.expAuth != authorization {
					t.Errorf("want\n%s\ngot\n%s", tc.expAuth, authorization)
				}

				rw.WriteHeader(http.StatusNoContent)

				close(tokenSeenCh)
				return
			}
			rw.WriteHeader(http.StatusBadRequest)
		}))
		defer oauthOrigin.Close()

		confPath := fmt.Sprintf("testdata/oauth2/%s", tc.configFile)
		shutdown, hook := newCouper(confPath, test.New(t))
		defer func() {
			if t.Failed() {
				for _, e := range hook.Entries {
					println(e.String())
				}
			}
			shutdown()
		}()

		req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
		helper.Must(err)

		req.Header.Set("X-Token-Endpoint", oauthOrigin.URL)

		hook.Reset()

		tokenSeenCh = make(chan struct{})

		req.URL.Path = "/"
		_, err = newClient().Do(req)
		helper.Must(err)

		timer := time.NewTimer(time.Second * 2)
		select {
		case <-timer.C:
			t.Error("OAuth2 request failed")
		case <-tokenSeenCh:
		}

		oauthOrigin.Close()
		shutdown()
	}
}

func TestOAuth2AccessControl(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			_ = req.ParseForm()
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100,
				"form_params": "` + req.PostForm.Encode() + `",
				"authorization": "` + req.Header.Get("Authorization") + `"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	type testCase struct {
		name          string
		filename      string
		path          string
		header        http.Header
		status        int
		params        string
		authorization string
		wantErrLog    string
	}

	st := "qeirtbnpetrbi"
	state := accesscontrol.Base64url_s256(st)

	for _, tc := range []testCase{
		{"no code, but error", "04_couper.hcl", "/cb?error=qeuboub", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: missing code query parameter; query='error=qeuboub"},
		{"no code; error handler", "05_couper.hcl", "/cb?error=qeuboub", http.Header{}, http.StatusBadRequest, "", "", "access control error: ac: missing code query parameter; query='error=qeuboub"},
		{"missing state param", "06_couper.hcl", "/cb?code=qeuboub", http.Header{"Cookie": []string{"st=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: missing state query parameter; query='code=qeuboub"},
		{"wrong state param", "06_couper.hcl", "/cb?code=qeuboub&state=wrong", http.Header{"Cookie": []string{"st=" + st}}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: 'wrong' (from query param) vs. 'qeirtbnpetrbi' (s256: 'oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ')"},
		{"code, state params, missing CSRF token", "06_couper.hcl", "/cb?code=qeuboub&state=" + state, http.Header{}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: 'oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ' (from query param) vs. '' (s256: '47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU')"},
		{"code; client_secret_basic; PKCE", "04_couper.hcl", "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_post", "05_couper.hcl", "/cb?code=qeuboub", http.Header{}, http.StatusOK, "client_id=foo&client_secret=etbinbp4in&code=qeuboub&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "", ""},
		{"code, state params", "06_couper.hcl", "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st}}, http.StatusOK, "code=qeuboub&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			shutdown, hook := newCouper("testdata/oauth2/"+tc.filename, test.New(t))
			defer shutdown()

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}
			req.Header.Set("X-Token-URL", oauthOrigin.URL)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				t.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
				return
			}

			tokenResBytes, err := ioutil.ReadAll(res.Body)
			var jData map[string]interface{}
			json.Unmarshal(tokenResBytes, &jData)
			if params, ok := jData["form_params"]; ok {
				if params != tc.params {
					t.Errorf("%q: expected params %s, got: %s", tc.name, tc.params, params)
					return
				}
			} else {
				if "" != tc.params {
					t.Errorf("%q: expected params %s, got no", tc.name, tc.params)
					return
				}
			}
			if authorization, ok := jData["authorization"]; ok {
				if authorization != tc.authorization {
					t.Errorf("%q: expected authorization %s, got: %s", tc.name, tc.authorization, authorization)
					return
				}
			} else {
				if "" != tc.authorization {
					t.Errorf("%q: expected authorization %s, got no", tc.name, tc.authorization)
					return
				}
			}

			message := getAccessControlMessages(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					t.Errorf("%q: Expected error log: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					t.Errorf("%q: Expected error log message: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			}
		})
	}
}
