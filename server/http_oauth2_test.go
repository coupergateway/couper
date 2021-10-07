package server_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"

	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/oauth2"
)

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)

	for i := range []int{0, 1, 2} {
		var seenCh, tokenSeenCh chan struct{}

		retries := 0

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/oauth2" {
				if accept := req.Header.Get("Accept"); accept != "application/json" {
					t.Errorf("expected Accept %q, got: %q", "application/json", accept)
				}
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
		defer shutdown()

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
				reqBody, _ := io.ReadAll(req.Body)
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
		defer shutdown()

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

func TestOAuth2_AccessControl(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	st := "qeirtbnpetrbi"
	state := oauth2.Base64urlSha256(st)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			if accept := req.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("expected Accept %q, got: %q", "application/json", accept)
			}
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
				idToken, _ := lib.CreateJWT("HS256", []byte("$e(rEt"), mapClaims, nil)
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
		} else if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"token_endpoint": "http://` + req.Host + `/token",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
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
		method        string
		path          string
		header        http.Header
		status        int
		params        string
		authorization string
		wantErrLog    string
	}

	for _, tc := range []testCase{
		{"wrong method", "04_couper.hcl", http.MethodPost, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: wrong method (POST)"},
		{"oidc: wrong method", "07_couper.hcl", http.MethodPost, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: wrong method (POST)"},
		{"no code, but error", "04_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"no code; error handler", "05_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusTeapot, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"oidc: no code; error handler", "10_couper.hcl", http.MethodGet, "/cb?error=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusTeapot, "", "", "access control error: ac: missing code query parameter; query=\"error=qeuboub\""},
		{"code, missing state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"st=qerbnr"}}, http.StatusForbidden, "", "", "access control error: ac: missing state query parameter; query=\"code=qeuboub\""},
		{"code, wrong state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=wrong", http.Header{"Cookie": []string{"st=" + st}}, http.StatusForbidden, "", "", "access control error: ac: state mismatch: \"wrong\" (from query param) vs. \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (verifier_value: \"qeirtbnpetrbi\")"},
		{"code, state param, wrong CSRF token", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: state mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from query param) vs. \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\" (verifier_value: \"qeirtbnpetrbi-wrong\")"},
		{"code, state param, missing CSRF token", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty verifier_value"},
		{"code, missing nonce", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing nonce claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"sub\":\"myself\"}'"},
		{"code, wrong nonce", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wn-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: nonce mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ-wrong\" (from nonce claim) vs. \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (verifier_value: \"qeirtbnpetrbi\")"},
		{"code, nonce, wrong CSRF token", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st + "-wrong"}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: nonce mismatch: \"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\" (from nonce claim) vs. \"Mj0ecDMNNzOwqUt1iFlY8TOTTKa17ISo8ARgt0pyb1A\" (verifier_value: \"qeirtbnpetrbi-wrong\")"},
		{"code, nonce, missing CSRF token", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{}, http.StatusForbidden, "", "", "access control error: ac: Empty verifier_value"},
		{"code, missing sub claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-msub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing sub claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\"}'"},
		{"code, sub mismatch", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wsub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: subject mismatch, in ID token \"me\", in userinfo response \"myself\""},
		{"code, missing exp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mexp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing exp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, missing iat claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-miat-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing iat claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"azp\":\"foo\", \"exp\":4e+09, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, missing azp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-mazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: missing azp claim in ID token, claims='jwt.MapClaims{\"aud\":[]interface {}{\"foo\", \"another-client-id\"}, \"exp\":4e+09, \"iat\":1000, \"iss\":\"https://authorization.server\", \"nonce\":\"oUuoMU0RFWI5itMBnMTt_TJ4SxxgE96eZFMNXSl63xQ\", \"sub\":\"myself\"}'"},
		{"code, wrong azp claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wazp-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: azp claim / client ID mismatch, azp = \"bar\", client ID = \"foo\""},
		{"code, missing iss claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-miss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, wrong iss claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-wiss-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token issuer is invalid: 'iss' value doesn't match expectation"},
		{"code, wrong aud claim", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-waud-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusForbidden, "", "", "access control error: ac: token response validation error: token audience is invalid: 'foo' wasn't found in aud claim"},
		{"code; client_secret_basic; PKCE", "04_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_post", "05_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}}, http.StatusOK, "client_id=foo&client_secret=etbinbp4in&code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "", ""},
		{"code, state param", "06_couper.hcl", http.MethodGet, "/cb?code=qeuboub&state=" + state, http.Header{"Cookie": []string{"st=" + st}}, http.StatusOK, "code=qeuboub&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code, nonce param", "07_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}}, http.StatusOK, "code=qeuboub-id&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; client_secret_basic; PKCE; relative redirect_uri", "08_couper.hcl", http.MethodGet, "/cb?code=qeuboub", http.Header{"Cookie": []string{"pkcecv=qerbnr"}, "X-Forwarded-Proto": []string{"https"}, "X-Forwarded-Host": []string{"www.example.com"}}, http.StatusOK, "code=qeuboub&code_verifier=qerbnr&grant_type=authorization_code&redirect_uri=https%3A%2F%2Fwww.example.com%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
		{"code; nonce param; relative redirect_uri", "09_couper.hcl", http.MethodGet, "/cb?code=qeuboub-id", http.Header{"Cookie": []string{"nnc=" + st}, "X-Forwarded-Proto": []string{"https"}, "X-Forwarded-Host": []string{"www.example.com"}}, http.StatusOK, "code=qeuboub-id&grant_type=authorization_code&redirect_uri=https%3A%2F%2Fwww.example.com%2Fcb", "Basic Zm9vOmV0YmluYnA0aW4=", ""},
	} {
		t.Run(tc.path[1:], func(subT *testing.T) {
			shutdown, hook := newCouperWithTemplate("testdata/oauth2/"+tc.filename, test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
			defer shutdown()

			helper := test.New(subT)

			req, err := http.NewRequest(tc.method, "http://back.end:8080"+tc.path, nil)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.status {
				subT.Errorf("%q: expected Status %d, got: %d", tc.name, tc.status, res.StatusCode)
			}

			tokenResBytes, err := io.ReadAll(res.Body)
			helper.Must(err)

			var jData map[string]interface{}
			json.Unmarshal(tokenResBytes, &jData)
			if params, ok := jData["form_params"]; ok {
				if params != tc.params {
					subT.Errorf("%q: expected params %s, got: %s", tc.name, tc.params, params)
				}
			} else {
				if tc.params != "" {
					subT.Errorf("%q: expected params %s, got no", tc.name, tc.params)
				}
			}
			if authorization, ok := jData["authorization"]; ok {
				if tc.authorization != authorization {
					subT.Errorf("%q: expected authorization %s, got: %s", tc.name, tc.authorization, authorization)
				}
			} else {
				if tc.authorization != "" {
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

func TestOAuth2_Locking(t *testing.T) {
	helper := test.New(t)
	client := test.NewHTTPClient()

	token := "token-"
	var oauthRequestCount uint32

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/oauth2" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		atomic.AddUint32(&oauthRequestCount, 1)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		n := fmt.Sprintf("%d", atomic.LoadUint32(&oauthRequestCount))
		body := []byte(`{
				"access_token": "` + token + n + `",
				"token_type": "bearer",
				"expires_in": 1.5
			}`)

		// Slow down token request
		time.Sleep(time.Second)

		_, werr := rw.Write(body)
		if werr != nil {
			t.Error(werr)
		}
	}))
	defer oauthOrigin.Close()

	ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/resource" {
			if auth := req.Header.Get("Authorization"); auth != "" {
				rw.Header().Set("Token", auth[len("Bearer "):])
				rw.WriteHeader(http.StatusNoContent)
			}

			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
	defer ResourceOrigin.Close()

	confPath := "testdata/oauth2/1_retries_couper.hcl"
	shutdown, hook := newCouperWithTemplate(
		confPath, test.New(t), map[string]interface{}{
			"asOrigin": oauthOrigin.URL,
			"rsOrigin": ResourceOrigin.URL,
		},
	)
	defer shutdown()

	req, rerr := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(rerr)

	hook.Reset()

	req.URL.Path = "/"

	var responses []*http.Response
	var wg sync.WaitGroup

	addLock := &sync.Mutex{}
	// Fire 5 requests in parallel...
	waitCh := make(chan struct{})
	errors := make(chan error, 5)
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			<-waitCh
			res, e := client.Do(req)
			if e != nil {
				errors <- e
				return
			}

			addLock.Lock()
			responses = append(responses, res)
			addLock.Unlock()

		}()
	}
	close(waitCh)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}

	for _, res := range responses {
		if res.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status NoContent, got: %d", res.StatusCode)
		}

		if token+"1" != res.Header.Get("Token") {
			t.Errorf("Invalid token given: want %s1, got: %s", token, res.Header.Get("Token"))
		}
	}

	if count := atomic.LoadUint32(&oauthRequestCount); count != 1 {
		t.Errorf("OAuth2 requests: want 1, got: %d", count)
	}

	t.Run("Lock is effective", func(st *testing.T) {
		// Wait until token has expired.
		time.Sleep(2 * time.Second)

		// Fetch new token.
		go func() {
			res, err := client.Do(req)
			if err != nil {
				st.Error(err)
				return
			}

			if token+"2" != res.Header.Get("Token") {
				st.Errorf("Received wrong token: want %s2, got: %s", token, res.Header.Get("Token"))
			}
		}()

		// Slow response due to lock
		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			st.Error(err)
			return
		}

		timeElapsed := time.Since(start)

		if token+"2" != res.Header.Get("Token") {
			st.Errorf("Received wrong token: want %s2, got: %s", token, res.Header.Get("Token"))
		}

		if timeElapsed < time.Second {
			st.Errorf("Response came too fast: dysfunctional lock?! (%s)", timeElapsed.String())
		}
	})

	t.Run("Mem store expiry", func(st *testing.T) {
		// Wait again until token has expired.
		time.Sleep(2 * time.Second)
		h := test.New(st)
		// Request fresh token and store in memstore
		res, err := client.Do(req)
		h.Must(err)

		if res.StatusCode != http.StatusNoContent {
			st.Errorf("Unexpected response status: want %d, got: %d", http.StatusNoContent, res.StatusCode)
		}

		if token+"3" != res.Header.Get("Token") {
			st.Errorf("Received wrong token: want %s3, got: %s", token, res.Header.Get("Token"))
		}

		if count := atomic.LoadUint32(&oauthRequestCount); count != 3 {
			st.Errorf("Unexpected number of OAuth2 requests: want 3, got: %d", count)
		}

		// Disconnect OAuth server
		oauthOrigin.Close()

		// Next request gets token from memstore
		res, err = client.Do(req)
		h.Must(err)
		if res.StatusCode != http.StatusNoContent {
			st.Errorf("Unexpected response status: want %d, got: %d", http.StatusNoContent, res.StatusCode)
		}

		if token+"3" != res.Header.Get("Token") {
			st.Errorf("Wrong token from mem store: want %s3, got: %s", token, res.Header.Get("Token"))
		}

		// Wait until token has expired. Next request accesses the OAuth server again.
		time.Sleep(2 * time.Second)
		res, err = newClient().Do(req)
		h.Must(err)
		if res.StatusCode != http.StatusBadGateway {
			st.Errorf("Unexpected response status: want %d, got: %d", http.StatusBadGateway, res.StatusCode)
		}
	})
}
