package server_test

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/internal/test"
)

func TestCustomLogs_Upstream(t *testing.T) {
	client := test.NewHTTPClient()

	shutdown, hook := newCouper("testdata/integration/logs/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		path  string
		expAL logrus.Fields
		expUL logrus.Fields
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
			logrus.Fields{"backend": string("couper test-backend")},
		},
		{
			"/jwt-valid",
			logrus.Fields{"jwt_regular": "GET", "server": "couper test-backend"},
			logrus.Fields{"backend": string("couper test-backend")},
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
			time.Sleep(200 * time.Millisecond)

			// Access log
			entries := hook.AllEntries()
			if len(entries) < 2 {
				st.Fatal("Expected logs, got nothing")
			}

			got, ok := entries[1].Data["custom"].(logrus.Fields)
			if !ok {
				st.Fatalf("expected\n%#v\ngot\n%#v", tc.expAL, got)
			}
			if !reflect.DeepEqual(tc.expAL, got) {
				st.Errorf("expected\n%#v\ngot\n%#v", tc.expAL, got)
			}

			// Upstream log
			got, ok = entries[0].Data["custom"].(logrus.Fields)
			if !ok {
				st.Fatalf("expected\n%#v\ngot\n%#v", tc.expUL, got)
			}
			if !cmp.Equal(tc.expUL, got) {
				st.Error(cmp.Diff(tc.expUL, got))
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
		{"saml with error-handler", "/saml/acs", nil, logrus.Fields{"saml_error": "GET", "saml_regular": "GET"}},
		{"saml with * error-handler", "/saml-wildcard/acs", nil, logrus.Fields{"saml_wildcard_error": "GET", "saml_regular": "GET"}},
		{"oidc with error-handler", "/oidc/cb", nil, logrus.Fields{"oidc_error": "GET", "oidc_regular": "GET"}},
		{"oidc with * error-handler", "/oidc-wildcard/cb", nil, logrus.Fields{"oidc_wildcard_error": "GET", "oidc_regular": "GET"}},
		{"file access", "/file.html", nil, logrus.Fields{"files": "GET"}},
		{"spa access", "/spa", nil, logrus.Fields{"spa": "GET"}},
		{"endpoint with error-handler", "/error-handler/endpoint",
			test.Header{"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.Qf0lkeZKZ3NJrYm3VdgiQiQ6QTrjCvISshD_q9F8GAM"},
			logrus.Fields{"error_handler": "GET", "jwt_regular": "GET"}},
	} {
		t.Run(tc.name, func(st *testing.T) {
			helper := test.New(st)
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080"+tc.path, nil)
			helper.Must(err)

			for k, v := range tc.header {
				req.Header.Set(k, v)
			}

			hook.Reset()
			_, err = client.Do(req)
			helper.Must(err)

			// Wait for logs
			time.Sleep(200 * time.Millisecond)

			// Access log
			entries := hook.AllEntries()
			if len(entries) == 0 {
				st.Errorf("expected log entries, got none")
				return
			}

			got, ok := entries[0].Data["custom"].(logrus.Fields)
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
