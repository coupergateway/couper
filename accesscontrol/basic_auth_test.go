package accesscontrol_test

import (
	b64 "encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	ac "github.com/coupergateway/couper/accesscontrol"
	couperErr "github.com/coupergateway/couper/errors"
)

func Test_NewBasicAuth(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	err := ba.Validate(req)
	if err != couperErr.Configuration {
		t.Errorf("Expected configuration error, got: %v", err)
	}

	type testCase struct {
		name, user, pass, file string
		expErrMsg              string
		shouldFail             bool
	}

	for _, tc := range []testCase{
		{"name", "user", "pass", "", "", false},
		{"name", "user", "", "", "", false},
		{"name", "", "pass", "", "", false},
		{"name", "", "", "", "", false},
		{"name", "user", "pass", "testdata/htpasswd", "", false},
		{"name", "", "", "testdata/htpasswd", "", false},
		{"name", "john", "pass", "testdata/htpasswd", "", false},
		{"name", "user", "pass", "file", "open file: no such file or directory", true},
		{"name", "user", "pass", "testdata/htpasswd_err_invalid", "parse error: invalid line: 1", true},
		{"name", "user", "pass", "testdata/htpasswd_err_too_long", "parse error: line length exceeded: 255 (line 1)", true},
		{"name", "user", "pass", "testdata/htpasswd_err_malformed", `parse error: malformed password for user "foo" (line 1)`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_multi", `multiple user: foo (line 2)`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_unsupported", "parse error: algorithm not supported (line 1)", true},
		{"name", "user", "pass", "testdata/htpasswd_err_argon2_time_zero", `parse error: malformed password for user "jack" (line 1): invalid argon2 parameter t: must be >= 1`, true},
	} {
		ba, err = ac.NewBasicAuth(tc.name, tc.user, tc.pass, tc.file, nil, nil)
		if tc.shouldFail && ba != nil {
			t.Error("Expected no successful basic auth creation")
		}

		if tc.shouldFail && err != nil && tc.expErrMsg != "" {
			if err.Error() != tc.expErrMsg {
				t.Errorf("Expected error message: %q, got: %q", tc.expErrMsg, err.Error())
			}
		} else if err != nil {
			t.Error(err)
		}
	}
}

// Test_NewBasicAuth_Argon2OverCapWarns asserts that htpasswd entries
// whose argon2 parameters exceed the recommended maxima still load
// (so an upgrade cannot break a running deployment) and that each
// over-cap parameter produces a startup warning.
func Test_NewBasicAuth_Argon2OverCapWarns(t *testing.T) {
	logger, hook := logrustest.NewNullLogger()
	logger.SetLevel(logrus.WarnLevel)

	ba, err := ac.NewBasicAuth("ac-name", "", "", "testdata/htpasswd_argon2_over_cap", nil, logrus.NewEntry(logger))
	if err != nil {
		t.Fatalf("expected over-cap entries to load with a warning, got error: %v", err)
	}
	if ba == nil {
		t.Fatal("expected a basic auth object")
	}

	var warnings []string
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			warnings = append(warnings, e.Message)
		}
	}
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings (m, t, p), got %d: %v", len(warnings), warnings)
	}

	for _, param := range []string{"m=", "t=", "p="} {
		found := false
		for _, w := range warnings {
			if strings.Contains(w, param) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a warning naming the %q parameter, got: %v", param, warnings)
		}
	}
}

func Test_BasicAuth_Validate(t *testing.T) {
	ba, err := ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", nil, nil)
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"", couperErr.AccessControl},
		{"Foo", couperErr.BasicAuth},
		{"Basic X", couperErr.BasicAuth},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("usr:pwd:foo")), couperErr.BasicAuth},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"bAsIc   " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"Asdfg " + b64.StdEncoding.EncodeToString([]byte("user:bass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:bass")), couperErr.BasicAuth},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-bass")), couperErr.BasicAuth},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("jack:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("jack:wrong")), couperErr.BasicAuth},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("jim:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("jim:wrong")), couperErr.BasicAuth},
	} {
		t.Run(testcase.headerValue, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", testcase.headerValue)
			err = ba.Validate(req)
			if testcase.expErr != nil && err.Error() != testcase.expErr.Error() {
				subT.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}

func Test_BasicAuth_ValidateCases(t *testing.T) {
	req := &http.Request{Header: make(http.Header)}

	ba1, err := ac.NewBasicAuth("name", "", "pass", "", nil, nil)
	if err != nil || ba1 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba2, err := ac.NewBasicAuth("name", "user", "", "", nil, nil)
	if err != nil || ba2 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba3, err := ac.NewBasicAuth("name", "", "", "", nil, nil)
	if err != nil || ba3 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba4, err := ac.NewBasicAuth("name", "", "", "testdata/htpasswd", nil, nil)
	if err != nil || ba4 == nil {
		t.Fatal("Expected a basic auth object")
	}

	type testCase struct {
		ba          *ac.BasicAuth
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{ba1, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), couperErr.BasicAuth},
		{ba1, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), couperErr.BasicAuth},
		{ba1, "Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), nil},
		{ba1, "Basic " + b64.StdEncoding.EncodeToString([]byte(":")), couperErr.BasicAuth},
		{ba2, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), couperErr.BasicAuth},
		{ba2, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), couperErr.BasicAuth},
		{ba2, "Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), couperErr.BasicAuth},
		{ba2, "Basic " + b64.StdEncoding.EncodeToString([]byte(":")), couperErr.BasicAuth},
		{ba3, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), couperErr.BasicAuth},
		{ba3, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), couperErr.BasicAuth},
		{ba3, "Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), couperErr.BasicAuth},
		{ba3, "Basic " + b64.StdEncoding.EncodeToString([]byte(":")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-pass")), nil},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte("john:")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte(":my-pass")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte(":")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), couperErr.BasicAuth},
		{ba4, "Basic " + b64.StdEncoding.EncodeToString([]byte(":")), couperErr.BasicAuth},
	} {
		t.Run(testcase.headerValue, func(subT *testing.T) {
			req.Header.Set("Authorization", testcase.headerValue)
			err = testcase.ba.Validate(req)
			if testcase.expErr != nil && !couperErr.Equals(err, testcase.expErr) {
				subT.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}
