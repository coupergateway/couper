package accesscontrol_test

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	ac "github.com/avenga/couper/accesscontrol"
)

func Test_NewBasicAuth(t *testing.T) {
	ba, err := ac.NewBasicAuth("name", "user", "pass", "", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "john", "pass", "testdata/htpasswd", "Basic")
	if ba == nil || err != nil {
		t.Errorf("Got unexpected error: '%s'", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "file", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "no such file or directory") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_invalid", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "invalid line") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_too_long", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "too long line") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_malformed", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "malformed") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_multi", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "multiple user") {
		t.Errorf("Got unexpected error: %s", err)
	}

	ba, err = ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd_err_unsupported", "Basic")
	if ba != nil || err == nil {
		t.Error("Got unexpected BasicAuth object")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "unsupported password algorithm") {
		t.Errorf("Got unexpected error: %s", err)
	}
}

func Test_Validate(t *testing.T) {
	var ba *ac.BasicAuth
	req := &http.Request{Header: make(http.Header)}

	if err := ba.Validate(req); err != ac.ErrorBasicAuthNotConfigured {
		t.Errorf("Expected NotConfigured error, got: %v", err)
	}

	ba, err := ac.NewBasicAuth("name", "user", "pass", "testdata/htpasswd", "Basic")
	if err != nil || ba == nil {
		t.Fatal("Expected a basic auth object")
	}

	var ebau *ac.ErrorBAUnauthorized

	type testCase struct {
		headerValue string
		expErr      error
	}

	for _, testcase := range []testCase{
		{"", ebau},
		{"Foo", ebau},
		{"Basic X", ebau},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("usr:pwd:foo")), ebau},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"bAsIc   " + b64.StdEncoding.EncodeToString([]byte("user:pass")), nil},
		{"Asdfg " + b64.StdEncoding.EncodeToString([]byte("user:bass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("user:bass")), ebau},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-pass")), nil},
		{"Basic " + b64.StdEncoding.EncodeToString([]byte("john:my-bass")), ebau},
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
