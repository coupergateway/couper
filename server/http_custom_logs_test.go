package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/logging"
	"github.com/coupergateway/couper/oauth2"
)

func TestCustomLogs_Upstream(t *testing.T) {
	t.Skip("TODO: stabilize")
	client := test.NewHTTPClient()

	shutdown, hook := newCouper("testdata/integration/logs/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path        string
		expAccess   logrus.Fields
		expUpstream logrus.Fields
	}

	hmacToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwic2NvcGUiOiJmb28gYmFyIiwiaWF0IjoxNTE2MjM5MDIyfQ.7wz7Z7IajfEpwYayfshag6tQVS0e0zZJyjAhuFC0L-E"

	for _, tc := range []testCase{
		{
			"/",
			logrus.Fields{
				"api":      "couper test-backend",
				"endpoint": "couper test-backend",
				"server":   "couper test-backend",
			},
			logrus.Fields{
				"array": []interface{}{
					float64(1),
					"couper test-backend",
					[]interface{}{
						float64(2),
						"couper test-backend",
					},
					logrus.Fields{"x": "X"},
				},
				"bool":   true,
				"float":  1.23,
				"int":    float64(123),
				"object": logrus.Fields{"a": "A", "b": "B", "c": float64(123)},
				"req":    "GET",
				"string": "couper test-backend",
			},
		},
		{
			"/backend",
			logrus.Fields{"api": "couper test-backend", "server": "couper test-backend"},
			logrus.Fields{"backend": "couper test-backend"},
		},
		{
			"/jwt-valid",
			logrus.Fields{"jwt_regular": "GET", "server": "couper test-backend"},
			logrus.Fields{"backend": "couper test-backend"},
		},
	} {
		t.Run(tc.path, func(st *testing.T) {
			helper := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			req.Header.Set("Authorization", "Bearer "+hmacToken)

			hook.Reset()
			_, err = client.Do(req)
			helper.Must(err)

			// Wait for logs
			time.Sleep(100 * time.Millisecond)

			entries := hook.AllEntries()
			var accessLog, upstreamLog logrus.Fields
			for _, entry := range entries {
				request := entry.Data["request"].(logging.Fields)
				path, _ := request["path"].(string)
				if entry.Data["type"] == "couper_access" {
					accessLog = entry.Data
				} else if entry.Data["type"] == "couper_backend" && path != "/jwks.json" {
					upstreamLog = entry.Data
				}
			}

			if accessLog == nil || upstreamLog == nil {
				st.Fatalf("expected logs, got access: %p, got upstream: %p", accessLog, upstreamLog)
			}

			customAccess, ok := accessLog["custom"].(logrus.Fields)
			if !ok {
				st.Fatal("expected access log custom field")
			}

			customUpstream, ok := upstreamLog["custom"].(logrus.Fields)
			if !ok {
				st.Fatal("expected upstream log custom field")
			}

			if !cmp.Equal(tc.expAccess, customAccess) {
				st.Error(cmp.Diff(tc.expAccess, customAccess))
			}

			if !cmp.Equal(tc.expUpstream, customUpstream) {
				st.Error(cmp.Diff(tc.expUpstream, customUpstream))
			}
		})
	}
}

func TestCustomLogs_Local(t *testing.T) {
	client := newClient()

	shutdown, hook := newCouper("testdata/integration/logs/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name   string
		path   string
		header test.Header
		exp    logrus.Fields
	}

	for _, tc := range []testCase{
		{"basic-auth", "/secure", nil, logrus.Fields{"error_handler": "GET"}},
		{"jwt with error-handler", "/jwt", nil, logrus.Fields{"jwt_error": "GET", "jwt_regular": "GET"}},
		{"jwt with * error-handler", "/jwt-wildcard", nil, logrus.Fields{"jwt_error_wildcard": "GET", "jwt_regular": "GET"}},
		{"oauth2 error-handler", "/oauth2cb?pkcecv=qerbnr&error=qeuboub", nil, logrus.Fields{"oauth2_error": "GET", "oauth2_regular": "GET"}},
		{"oauth2 * error-handler", "/oauth2cb-wildcard?pkcecv=qerbnr&error=qeuboub", nil, logrus.Fields{"oauth2_wildcard_error": "GET", "oauth2_regular": "GET"}},
		{"saml with saml2 error-handler", "/saml-saml2/acs", nil, logrus.Fields{"saml_saml2_error": "GET", "saml_regular": "GET"}},
		{"saml with saml error-handler", "/saml-saml/acs", nil, logrus.Fields{"saml_saml_error": "GET", "saml_regular": "GET"}},
		{"saml with * error-handler", "/saml-wildcard/acs", nil, logrus.Fields{"saml_wildcard_error": "GET", "saml_regular": "GET"}},
		{"oidc with error-handler", "/oidc/cb", nil, logrus.Fields{"oidc_error": "GET", "oidc_regular": "GET"}},
		{"oidc with * error-handler", "/oidc-wildcard/cb", nil, logrus.Fields{"oidc_wildcard_error": "GET", "oidc_regular": "GET"}},
		{"file access", "/file.html", nil, logrus.Fields{"files": "GET"}},
		{"spa access", "/spa", nil, logrus.Fields{"spa": "GET"}},
		{"endpoint with error-handler", "/error-handler/endpoint",
			test.Header{"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.Qf0lkeZKZ3NJrYm3VdgiQiQ6QTrjCvISshD_q9F8GAM"},
			logrus.Fields{"error_handler": "GET", "jwt_regular": "GET"}},
		{"endpoint standard", "/standard", nil, logrus.Fields{
			"server": "couper test-backend",
			"item-1": "item1",
			"item-2": "item1",
		}},
		{"endpoint sequence", "/sequence", nil, logrus.Fields{
			"server":     "couper test-backend",
			"seq-item-1": "item1",
			"seq-item-2": "item1",
		}},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()

			helper := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v)
			}

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			// Wait for logs
			time.Sleep(time.Second / 5)

			// Access log
			entries := hook.AllEntries()
			if len(entries) == 0 {
				st.Errorf("expected log entries, got none")
				return
			}

			var accessLogEntry *logrus.Entry
			for _, e := range entries {
				if e.Data["type"] == "couper_access" {
					accessLogEntry = e
					break
				}
			}
			if accessLogEntry == nil {
				st.Fatal("Expected access log entry")
			}

			got, ok := accessLogEntry.Data["custom"].(logrus.Fields)
			if !ok {
				st.Fatal("expected custom log field, got none")
			}

			if !reflect.DeepEqual(tc.exp, got) {
				st.Error(cmp.Diff(tc.exp, got))
			}
		})
	}
}

func TestCustomLogs_Merge(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/logs/02_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)

	// Wait for logs
	time.Sleep(200 * time.Millisecond)

	exp := logrus.Fields{
		"api":      true,
		"endpoint": true,
		"l1":       "endpoint",
		"l2":       []interface{}{"server", "api", "endpoint"},
		"l3":       []interface{}{"endpoint"},
		"server":   true,
	}

	// Access log
	got, ok := hook.AllEntries()[0].Data["custom"].(logrus.Fields)
	if !ok {
		t.Fatalf("expected\n%#v\ngot\n%#v", exp, got)
	}
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("expected\n%#v\ngot\n%#v", exp, got)
	}
}

func TestCustomLogs_EvalError(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/logs/04_couper.hcl", test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	hook.Reset()
	_, err = client.Do(req)
	helper.Must(err)
}

func TestCustomLogs_OIDCBackendResponse(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	st := "qeirtbnpetrbi"
	state := oauth2.Base64urlSha256(st)

	oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			body := []byte(`{
			"issuer": "https://authorization.server",
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"jwks_uri": "http://` + req.Host + `/jwks",
			"token_endpoint": "http://` + req.Host + `/token",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
			_, _ = rw.Write(body)
			return
		} else if req.URL.Path == "/jwks" {
			jsonBytes, err := os.ReadFile("testdata/integration/files/jwks.json")
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			b := bytes.NewBuffer(jsonBytes)
			_, _ = b.WriteTo(rw)
			return
		} else if req.URL.Path == "/token" {
			_ = req.ParseForm()
			rw.Header().Set("Content-Type", "application/json")

			nonce := state
			mapClaims := jwt.MapClaims{
				"aud":   []string{"foo"},
				"iss":   "https://authorization.server",
				"iat":   1000,
				"exp":   4000000000,
				"sub":   "myself",
				"azp":   "foo",
				"nonce": nonce,
			}
			keyBytes, err := os.ReadFile("testdata/integration/files/pkcs8.key")
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			idToken, err := lib.CreateJWT("RS256", key, mapClaims, map[string]interface{}{"kid": "rs256"})
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100,
				"id_token": "` + idToken + `"
			}`)
			_, _ = rw.Write(body)
			return
		} else if req.URL.Path == "/userinfo" {
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write([]byte(`{"sub": "myself"}`))
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	}))
	defer oauthOrigin.Close()

	shutdown, hook, err := newCouperWithTemplate("testdata/integration/logs/05_couper.hcl", test.New(t), map[string]interface{}{"asOrigin": oauthOrigin.URL})
	helper.Must(err)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://back.end:8080/cb?code=qeuboub", nil)
	helper.Must(err)
	req.Header.Set("Cookie", "nnc="+st)

	hook.Reset()
	res, err := client.Do(req)
	helper.Must(err)
	helper.Must(res.Body.Close())

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got: %d", http.StatusNoContent, res.StatusCode)
	}

	// Wait for logs
	time.Sleep(time.Second / 5)

	entries := hook.AllEntries()

	// Find upstream log entries for token request (not jwks or openid-configuration)
	var tokenUpstreamLog logrus.Fields
	for _, entry := range entries {
		if entry.Data["type"] != "couper_backend" {
			continue
		}
		r, ok := entry.Data["request"].(logging.Fields)
		if !ok {
			continue
		}
		path, _ := r["path"].(string)
		if strings.HasSuffix(path, "/token") {
			tokenUpstreamLog = entry.Data
			break
		}
	}

	if tokenUpstreamLog == nil {
		t.Fatal("expected upstream log entry for token request")
	}

	// Verify the custom log fields from backend_response.json_body
	customUpstream, ok := tokenUpstreamLog["custom"].(logrus.Fields)
	if !ok {
		t.Fatal("expected upstream custom log field for token request")
	}

	exp := logrus.Fields{
		"token_type": "bearer",
	}
	if !cmp.Equal(exp, customUpstream) {
		t.Error(cmp.Diff(exp, customUpstream))
	}

	// Verify the request name is set (not <nil>)
	requestFields, _ := tokenUpstreamLog["request"].(logging.Fields)
	if requestFields != nil {
		name, _ := requestFields["name"].(string)
		if name == "" {
			t.Errorf("expected non-empty request name for token upstream log, got: %q", name)
		}
	}
}
