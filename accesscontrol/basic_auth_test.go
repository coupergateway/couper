package accesscontrol_test

import (
	b64 "encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

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
		{"name", "user", "pass", "testdata/htpasswd_err_too_long", "parse error: line length exceeded: 255", true},
		{"name", "user", "pass", "testdata/htpasswd_err_malformed", `parse error: malformed password for user: foo`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_multi", `multiple user: foo`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_unsupported", "parse error: algorithm not supported", true},
	} {
		ba, err = ac.NewBasicAuth(tc.name, tc.user, tc.pass, tc.file)
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

func Test_BasicAuth_Validate(t *testing.T) {
	ba, err := ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd")
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

	ba1, err := ac.NewBasicAuth("name", "", "pass", "")
	if err != nil || ba1 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba2, err := ac.NewBasicAuth("name", "user", "", "")
	if err != nil || ba2 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba3, err := ac.NewBasicAuth("name", "", "", "")
	if err != nil || ba3 == nil {
		t.Fatal("Expected a basic auth object")
	}
	ba4, err := ac.NewBasicAuth("name", "", "", "testdata/htpasswd")
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
