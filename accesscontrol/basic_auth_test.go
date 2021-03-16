package accesscontrol_test

import (
	b64 "encoding/base64"
	"errors"
	"net/http"
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
)

func Test_NewBasicAuth(t *testing.T) {
	type testCase struct {
		name, user, pass, file, realm string
		expErrMsg                     string
		shouldFail                    bool
	}

	for _, tc := range []testCase{
		{"name", "user", "pass", "", "Basic", "", false},
		{"name", "user", "", "", "Basic", "", false},
		{"name", "", "pass", "", "Basic", "", false},
		{"name", "", "", "", "Basic", "", false},
		{"name", "user", "pass", "testdata/htpasswd", "Basic", "", false},
		{"name", "john", "pass", "testdata/htpasswd", "Basic", "", false},
		{"name", "user", "pass", "file", "Basic", "open file: no such file or directory", true},
		{"name", "user", "pass", "testdata/htpasswd_err_invalid", "Basic", "basic auth ht parse error: invalidLine: testdata/htpasswd_err_invalid:1", true},
		{"name", "user", "pass", "testdata/htpasswd_err_too_long", "Basic", "basic auth ht parse error: lineTooLong: testdata/htpasswd_err_too_long:1", true},
		{"name", "user", "pass", "testdata/htpasswd_err_malformed", "Basic", `basic auth ht parse error: malformedPassword: testdata/htpasswd_err_malformed:1: user "foo"`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_multi", "Basic", `basic auth ht parse error: multipleUser: testdata/htpasswd_err_multi:2: "foo"`, true},
		{"name", "user", "pass", "testdata/htpasswd_err_unsupported", "Basic", "basic auth ht parse error: notSupported: testdata/htpasswd_err_unsupported:1: unknown password algorithm", true},
	} {
		ba, err := ac.NewBasicAuth(tc.name, tc.user, tc.pass, tc.file, tc.realm)
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

func Test_BA_Validate(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", "Basic")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	var authError *ac.BasicAuthError

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"", authError},
		{"Foo", authError},
		{"Basic X", authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("usr:pwd:foo")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"bAsIc   " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"Asdfg " + b64.StdEncoding.EncodeToString([]byte("user:bass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:bass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-bass")), authError},
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

func Test_BA_ValidateEmptyUser(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "", "pass", "", "Basic")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	var authError *ac.BasicAuthError

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), authError},
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

func Test_BA_ValidateEmptyPassword(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "user", "", "", "Basic")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	var authError *ac.BasicAuthError

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), authError},
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

func Test_BA_ValidateEmptyUserPassword(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "", "", "", "Basic")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	var authError *ac.BasicAuthError

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":pass")), authError},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte(":")), authError},
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
