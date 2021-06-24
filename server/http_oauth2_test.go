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

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/eval/lib"
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
		shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL, "rsOrigin": ResourceOrigin.URL})
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
		shutdown, hook := newCouperWithTemplate(confPath, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
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

	st := "qeirtbnpetrbi"
	state := accesscontrol.Base64urlSha256(st)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			_ = req.ParseForm()
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			code := req.PostForm.Get("code")
			idTokenToAdd := ""
			if strings.HasSuffix(code, "-id") {
				nonce := state
				mapClaims := jwt.MapClaims{"aud": []string{"foo", "another-client-id"}}
				if !strings.HasSuffix(code, "-miss-id") {
					if strings.HasSuffix(code, "-wiss-id") {
						mapClaims["iss"] = "https://malicious.authorization.server"
					} else {
						mapClaims["iss"] = "https://authorization.server"
					}
				}
				if !strings.HasSuffix(code, "-miat-id") {
					// 1970-01-01 00:16:40 +0000 UTC
					mapClaims["iat"] = 1000
				}
				if !strings.HasSuffix(code, "-mexp-id") {
					// 2096-10-02 07:06:40 +0000 UTC
					mapClaims["exp"] = 4000000000
				}
				if !strings.HasSuffix(code, "-msub-id") {
					if strings.HasSuffix(code, "-wsub-id") {
						mapClaims["sub"] = "me"
					} else {
						mapClaims["sub"] = "myself"
					}
				}
				if strings.HasSuffix(code, "-waud-id") {
					mapClaims["aud"] = "another-client-id"
				}
				if strings.HasSuffix(code, "-wazp-id") {
					mapClaims["azp"] = "bar"
				} else if !strings.HasSuffix(code, "-mazp-id") {
					mapClaims["azp"] = "foo"
				}
				if strings.HasSuffix(code, "-wn-id") {
					nonce = nonce + "-wrong"
				}
				if !strings.HasSuffix(code, "-mn-id") {
					mapClaims["nonce"] = nonce
				}
				idToken, _ := lib.CreateJWT("HS256", []byte("$e(rEt"), mapClaims)
				idTokenToAdd = `"id_token":"` + idToken + `",
				`
			}

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100,
				` + idTokenToAdd +
				`"form_params": "` + req.PostForm.Encode() + `",
				"authorization": "` + req.Header.Get("Authorization") + `"
			}`)
			_, werr := rw.Write(body)
			helper.Must(werr)

			return
		} else if req.URL.Path == "/userinfo" {
			body := []byte(`{"sub": "myself"}`)
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

	for _, tc := range []testCase{
		{"no code, but error", "04_couper.hcl", "/cb?error=qeuboub", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"no code; error handler", "05_couper.hcl", "/cb?error=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusBadRequest, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"code, missing state param", "06_couper.hcl", "/cb?code=qeuboub", http.Header{"Cookie": []string{"st=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: missing state query parameter; query=\"code=qeuboub\""},
		{"code, wrong state param", "06_couper.hcl", "/cb?code=qeuboub&state=wrong", http.Header{"Cookie": []string{"st=" + st}}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: \"wrong\" (from query param) vs. \"qeirtbnpetrbi\" (s256: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\")"},
		{"code, state param, wrong CSRF token", "06_couper.hcl", "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from query param) vs. \"qeirtbnpetrbi-wrong\" (s256: \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\")"},
		{"code, state param, missing CSRF token", "06_couper.hcl", "/cb?code=qeuboub&state=" + state, http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty CSRF token_value"},
		{"code, missing nonce", "07_couper.hcl", "/cb?code=qeuboub-mn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: missing nonce claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"sub\":\"myself\"}'"},
		{"code, wrong nonce", "07_couper.hcl", "/cb?code=qeuboub-wn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ-wrong\" (from nonce claim) vs. \"qeirtbnpetrbi\" (s256: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\")"},
		{"code, nonce, wrong CSRF token", "07_couper.hcl", "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: CSRF token mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from nonce claim) vs. \"qeirtbnpetrbi-wrong\" (s256: \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\")"},
		{"code, nonce, missing CSRF token", "07_couper.hcl", "/cb?code=qeuboub-id", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty CSRF token_value"},
		{"code, missing sub claim", "07_couper.hcl", "/cb?code=qeuboub-msub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: missing sub claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\"}'"},
		{"code, sub mismatch", "07_couper.hcl", "/cb?code=qeuboub-wsub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: subject mismatch, in ID token \"me\", in userinfo response \"myself\""},
		{"code, missing exp claim", "07_couper.hcl", "/cb?code=qeuboub-mexp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: missing exp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, missing iat claim", "07_couper.hcl", "/cb?code=qeuboub-miat-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: missing iat claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, missing azp claim", "07_couper.hcl", "/cb?code=qeuboub-mazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: missing azp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, wrong azp claim", "07_couper.hcl", "/cb?code=qeuboub-wazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: azp claim / client ID mismatch, azp = \"bar\", client ID = \"foo\""},
		{"code, missing iss claim", "07_couper.hcl", "/cb?code=qeuboub-miss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, wrong iss claim", "07_couper.hcl", "/cb?code=qeuboub-wiss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, wrong aud claim", "07_couper.hcl", "/cb?code=qeuboub-waud-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token audience is invalid: 'foo' wasn't found in aud claim"},
		{"code; client_secret_basic; PKCE", "04_couper.hcl", "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_post", "05_couper.hcl", "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "client_id=foo&client_secret=etbinbp4in&code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "", ""},
		{"code, state param", "06_couper.hcl", "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st}}, http.StatusOK, "code=qeuboub&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code, nonce param", "07_couper.hcl", "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusOK, "code=qeuboub-id&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			shutdown, hook := newCouperWithTemplate("testdata/oauth2/"+tc.filename, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
			defer shutdown()

			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				subT.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
			}

			tokenResBytes, err := ioutil.ReadAll(res.Body)
			var jData map[string]interface{}
			json.Unmarshal(tokenResBytes, &jData)
			if params, ok := jData["form_params"]; ok {
				if params != tc.params {
					subT.Errorf("%q: expected params %s, got: %s", tc.name, tc.params, params)
				}
			} else {
				if "" != tc.params {
					subT.Errorf("%q: expected params %s, got no", tc.name, tc.params)
				}
			}
			if authorization, ok := jData["authorization"]; ok {
				if authorization != tc.authorization {
					subT.Errorf("%q: expected authorization %s, got: %s", tc.name, tc.authorization, authorization)
				}
			} else {
				if "" != tc.authorization {
					subT.Errorf("%q: expected authorization %s, got no", tc.name, tc.authorization)
				}
			}

			message := getAccessControlMessages(hook)
			if tc.wantErrLog == "" {
				if message != "" {
					subT.Errorf("%q: Expected error log: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			} else {
				if !strings.HasPrefix(message, tc.wantErrLog) {
					subT.Errorf("%q: Expected error log message: %q, actual: %#v", tc.name, tc.wantErrLog, message)
				}
			}
		})
	}
}
