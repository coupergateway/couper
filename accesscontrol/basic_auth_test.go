package accesscontrol_test

import (
	b64 "encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
	couperErr "github.com/avenga/couper/errors"
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
		{"name", "john", "pass", "testdata/htpasswd", "", false},
		{"name", "user", "pass", "file", "name: open file: no such file or directory", true},
		{"name", "user", "pass", "testdata/htpasswd_err_invalid", "name: parse error: invalid line: 1", true},
		{"name", "user", "pass", "testdata/htpasswd_err_too_long", "name: parse error: line length exceeded: 255", true},
		{"name", "user", "pass", "testdata/htpasswd_err_malformed", `name: parse error: malformed password for user: foo`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_multi", `name: multiple user: foo`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_unsupported", "name: parse error: algorithm not supported", true},
	} {
		ba, err = ac.NewBasicAuth(tc.name, tc.user, tc.pass, tc.file)
		if tc.shouldFail && ba != nil {
			t.Error("Expected no successful basic auth creation")
		}

		if tc.shouldFail && err != nil && tc.expErrMsg != "" {
			if err.Error() != couperErr.Configuration.Label("name").Error() {
				t.Errorf("Expected a configuration error")
			}
			gerr := err.(couperErr.GoError)

			if gerr.GoError() != tc.expErrMsg {
				t.Errorf("Expected error message: %q, got: %q", tc.expErrMsg, gerr.GoError())
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
		{"", ac.BasicAuthError},
		{"Foo", ac.BasicAuthError},
		{"Basic X", ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("usr:pwd:foo")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"bAsIc   " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"Asdfg " + b64.StdEncoding.EncodeToString([]byte("user:bass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:bass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-bass")), ac.BasicAuthError},
	} {
		t.Run(testcase.headerValue, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", testcase.headerValue)
			err = ba.Validate(req)
			if testcase.expErr != nil && !errors.As(err, &testcase.expErr) {
				t.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}

func Test_BasicAuth_ValidateEmptyUser(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != couperErr.Configuration {
		t.Errorf("Expected configuration error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "", "pass", "")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), ac.BasicAuthError},
	} {
		t.Run(testcase.headerValue, func(t *testing.T) {
			req.Header.Set("Authorization", testcase.headerValue)
			err := ba.Validate(req)
			if testcase.expErr != nil && !errors.As(err, &testcase.expErr) {
				t.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}

func Test_BasicAuth_ValidateEmptyPassword(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != couperErr.Configuration {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "user", "", "")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), ac.BasicAuthError},
	} {
		t.Run(testcase.headerValue, func(t *testing.T) {
			req.Header.Set("Authorization", testcase.headerValue)
			err := ba.Validate(req)
			if testcase.expErr != nil && !errors.As(err, &testcase.expErr) {
				t.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}

func Test_BasicAuth_ValidateEmptyUserPassword(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != couperErr.Configuration {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "", "", "")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), ac.BasicAuthError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), ac.BasicAuthError},
	} {
		t.Run(testcase.headerValue, func(t *testing.T) {
			req.Header.Set("Authorization", testcase.headerValue)
			err := ba.Validate(req)
			if testcase.expErr != nil && !errors.As(err, &testcase.expErr) {
				t.Errorf("Expected Unauthorized error, got: %v", err)
			}
		})
	}
}
